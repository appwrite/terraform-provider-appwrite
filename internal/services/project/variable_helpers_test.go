package project

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// variableServer answers every request with a variable and keeps the requests
// it received.
func variableServer(t *testing.T) (*httptest.Server, *[]capturedRequest) {
	t.Helper()

	var requests []capturedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		requests = append(requests, capturedRequest{method: r.Method, path: r.URL.Path, body: body})

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"$id": "variable-id", "key": "API_URL", "secret": true})
	}))
	t.Cleanup(server.Close)
	return server, &requests
}

// TestVariableRoutes checks the routes and arguments that reach the server.
// The SDK's project service also holds keys, platforms and policies, so a call
// wired to the wrong method would otherwise only show up against a server.
func TestVariableRoutes(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		secret types.Bool
		// wantSecret is the secret argument the server must receive, or nil
		// when none may be sent.
		wantSecret any
	}{
		// Unset in the configuration: the server's default applies on create.
		"secret unknown": {secret: types.BoolUnknown(), wantSecret: nil},
		"secret null":    {secret: types.BoolNull(), wantSecret: nil},
		"secret true":    {secret: types.BoolValue(true), wantSecret: true},
		"secret false":   {secret: types.BoolValue(false), wantSecret: false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			server, requests := variableServer(t)
			client := appwrite.NewProject(appwrite.NewClient(appwrite.WithEndpoint(server.URL+"/v1"), appwrite.WithProject("project-id")))

			plan := variableResourceModel{
				ID:     types.StringValue("variable-id"),
				Key:    types.StringValue("API_URL"),
				Secret: tc.secret,
			}
			if _, err := createVariable(client, plan, "first"); err != nil {
				t.Fatalf("create: %v", err)
			}
			if _, err := updateVariable(client, plan, "second"); err != nil {
				t.Fatalf("update: %v", err)
			}

			if len(*requests) != 2 {
				t.Fatalf("got %d requests, want 2", len(*requests))
			}
			for n, want := range []struct {
				method, path, value string
			}{
				{http.MethodPost, "/v1/project/variables", "first"},
				{http.MethodPut, "/v1/project/variables/variable-id", "second"},
			} {
				got := (*requests)[n]
				if got.method != want.method || got.path != want.path {
					t.Errorf("request %d: got %s %s, want %s %s", n, got.method, got.path, want.method, want.path)
				}
				if got.body["key"] != "API_URL" || got.body["value"] != want.value {
					t.Errorf("request %d: body %v, want key API_URL and value %q", n, got.body, want.value)
				}
				secret, sent := got.body["secret"]
				switch {
				case tc.wantSecret == nil && sent:
					t.Errorf("request %d: secret = %v sent, want it left to the server", n, secret)
				case tc.wantSecret != nil && secret != tc.wantSecret:
					t.Errorf("request %d: secret = %v, want %v", n, secret, tc.wantSecret)
				}
			}
			// A server-generated ID: the resource has no variable_id argument.
			if id, _ := (*requests)[0].body["variableId"].(string); id == "" || id == "variable-id" {
				t.Errorf("create sent variableId %q, want a fresh unique ID", id)
			}
		})
	}
}

func TestMapVariableToState(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		secret    bool
		served    string
		prior     types.String
		wantValue types.String
	}{
		// The API returns a secret's value empty; keep what was configured.
		"secret keeps configured value": {secret: true, served: "", prior: types.StringValue("configured"), wantValue: types.StringValue("configured")},
		// A non-secret value is returned in full, so drift is visible,
		// including a value emptied outside Terraform.
		"non-secret refreshes value":      {secret: false, served: "changed", prior: types.StringValue("configured"), wantValue: types.StringValue("changed")},
		"non-secret refreshes to empty":   {secret: false, served: "", prior: types.StringValue("configured"), wantValue: types.StringValue("")},
		"value_wo stays out of state":     {secret: false, served: "from-value-wo", prior: types.StringNull(), wantValue: types.StringNull()},
		"secret value_wo stays out":       {secret: true, served: "", prior: types.StringNull(), wantValue: types.StringNull()},
		"import leaves value to the user": {secret: false, served: "served", prior: types.StringNull(), wantValue: types.StringNull()},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			model := variableResourceModel{Value: tc.prior}
			mapVariableToState(&models.Variable{
				Id:        "variable-id",
				Key:       "API_URL",
				Value:     tc.served,
				Secret:    tc.secret,
				CreatedAt: "2026-11-04T00:00:00.000+00:00",
				UpdatedAt: "2026-11-04T00:00:01.000+00:00",
			}, &model)

			if !model.Value.Equal(tc.wantValue) {
				t.Errorf("value = %v, want %v", model.Value, tc.wantValue)
			}
			if !model.Secret.Equal(types.BoolValue(tc.secret)) {
				t.Errorf("secret = %v, want %v", model.Secret, tc.secret)
			}
			if model.ID.ValueString() != "variable-id" || model.Key.ValueString() != "API_URL" {
				t.Errorf("id/key = %v/%v", model.ID, model.Key)
			}
		})
	}
}

func TestSecretRequiresReplace(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		state, plan types.Bool
		want        bool
	}{
		"secret to non-secret":     {state: types.BoolValue(true), plan: types.BoolValue(false), want: true},
		"non-secret to secret":     {state: types.BoolValue(false), plan: types.BoolValue(true), want: false},
		"unchanged secret":         {state: types.BoolValue(true), plan: types.BoolValue(true), want: false},
		"unchanged non-secret":     {state: types.BoolValue(false), plan: types.BoolValue(false), want: false},
		"secret, plan unknown":     {state: types.BoolValue(true), plan: types.BoolUnknown(), want: false},
		"secret, removed from cfg": {state: types.BoolValue(true), plan: types.BoolNull(), want: false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			resp := &boolplanmodifier.RequiresReplaceIfFuncResponse{}
			secretRequiresReplace(context.Background(), planmodifier.BoolRequest{StateValue: tc.state, PlanValue: tc.plan}, resp)
			if resp.RequiresReplace != tc.want {
				t.Errorf("RequiresReplace = %v, want %v", resp.RequiresReplace, tc.want)
			}
		})
	}
}

func TestParseVariableImportID(t *testing.T) {
	t.Parallel()

	for importID, want := range map[string]struct {
		projectID, variableID string
		err                   bool
	}{
		"variable-id":            {variableID: "variable-id"},
		"project-id/variable-id": {projectID: "project-id", variableID: "variable-id"},
		"":                       {err: true},
		"/variable-id":           {err: true},
		"project-id/":            {err: true},
		"a/b/c":                  {err: true},
	} {
		projectID, variableID, err := parseVariableImportID(importID)
		if (err != nil) != want.err {
			t.Errorf("parseVariableImportID(%q) error = %v, want error %v", importID, err, want.err)
			continue
		}
		if projectID != want.projectID || variableID != want.variableID {
			t.Errorf("parseVariableImportID(%q) = %q, %q, want %q, %q", importID, projectID, variableID, want.projectID, want.variableID)
		}
	}
}
