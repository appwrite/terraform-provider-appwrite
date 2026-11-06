package user

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/sdk-for-go/v7/users"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestValidateTargetIdentifier(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		providerType string
		identifier   string
		valid        bool
	}{
		{"email", "someone@example.com", true},
		{"email", "Someone@Example.COM", true},
		{"email", "first.last+tag@sub.example.co.uk", true},
		{"email", "not-an-email", false},
		{"email", "@example.com", false},
		{"email", "someone@", false},
		{"email", "a@b@example.com", false},
		{"sms", "+14155550100", true},
		{"sms", "+3161234567", true},
		{"sms", "+123456789012345", true},
		{"sms", "14155550100", false},
		{"sms", "+04155550100", false},
		{"sms", "+1415555", true},
		{"sms", "+141555", false},
		{"sms", "+1234567890123456", false},
		{"sms", "+1 415 555 0100", false},
		// Push tokens are opaque; anything the length check admits is fine.
		{"push", "any-device-token", true},
		{"push", "not@an@email", true},
	} {
		diags := validateTargetIdentifier(tc.providerType, tc.identifier)
		if got := !diags.HasError(); got != tc.valid {
			t.Errorf("%s %q: valid = %v, want %v (%v)", tc.providerType, tc.identifier, got, tc.valid, diags)
		}
	}
}

func TestTargetIDPattern(t *testing.T) {
	t.Parallel()

	for id, valid := range map[string]bool{
		"target-1":   true,
		"Target.ID_": true,
		"6aeb1c4a64": true,
		"-leading":   false,
		".leading":   false,
		"_leading":   false,
		"has space":  false,
		"unique()":   false,
		"emoji-🙂":    false,
	} {
		if got := targetIDPattern.MatchString(id); got != valid {
			t.Errorf("%q: matches = %v, want %v", id, got, valid)
		}
	}
}

func TestIsContactTarget(t *testing.T) {
	t.Parallel()

	user := &models.User{Email: "someone@example.com", Phone: "+14155550100"}
	for _, tc := range []struct {
		name   string
		user   *models.User
		target models.Target
		want   bool
	}{
		{"email target for the user's email", user, models.Target{ProviderType: "email", Identifier: "someone@example.com"}, true},
		{"sms target for the user's phone", user, models.Target{ProviderType: "sms", Identifier: "+14155550100"}, true},
		{"another email target", user, models.Target{ProviderType: "email", Identifier: "other@example.com"}, false},
		{"another sms target", user, models.Target{ProviderType: "sms", Identifier: "+14155550199"}, false},
		{"push target", user, models.Target{ProviderType: "push", Identifier: "someone@example.com"}, false},
		{"type must match too", user, models.Target{ProviderType: "sms", Identifier: "someone@example.com"}, false},
		{"user without email", &models.User{}, models.Target{ProviderType: "email", Identifier: ""}, false},
		// Identifiers are stored as written, so only the exact address is the
		// user's own.
		{"email in another case", user, models.Target{ProviderType: "email", Identifier: "Someone@example.com"}, false},
	} {
		if got := isContactTarget(tc.user, &tc.target); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestMapTargetToState(t *testing.T) {
	t.Parallel()

	providerID := "provider-1"
	for _, tc := range []struct {
		name         string
		target       models.Target
		wantName     types.String
		wantProvider types.String
	}{
		{
			name:         "unset name and provider are null",
			target:       models.Target{Id: "t", UserId: "u", ProviderType: "email", Identifier: "a@example.com"},
			wantName:     types.StringNull(),
			wantProvider: types.StringNull(),
		},
		{
			name:         "set name and provider are kept",
			target:       models.Target{Id: "t", UserId: "u", ProviderType: "push", Identifier: "tok", Name: "Pixel", ProviderId: &providerID},
			wantName:     types.StringValue("Pixel"),
			wantProvider: types.StringValue("provider-1"),
		},
	} {
		var model targetResourceModel
		mapTargetToState(&tc.target, &model)
		if !model.Name.Equal(tc.wantName) {
			t.Errorf("%s: name = %v, want %v", tc.name, model.Name, tc.wantName)
		}
		if !model.ProviderID.Equal(tc.wantProvider) {
			t.Errorf("%s: provider_id = %v, want %v", tc.name, model.ProviderID, tc.wantProvider)
		}
		if model.UserID.ValueString() != "u" || model.ID.ValueString() != "t" {
			t.Errorf("%s: ids = %v/%v", tc.name, model.UserID, model.ID)
		}
	}
}

func TestRequiresReplaceIfRemoved(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		state types.String
		plan  types.String
		want  bool
	}{
		{"removed", types.StringValue("x"), types.StringNull(), true},
		{"changed", types.StringValue("x"), types.StringValue("y"), false},
		{"added", types.StringNull(), types.StringValue("y"), false},
		{"unknown", types.StringValue("x"), types.StringUnknown(), false},
		{"absent", types.StringNull(), types.StringNull(), false},
	} {
		req := planmodifier.StringRequest{StateValue: tc.state, PlanValue: tc.plan, ConfigValue: tc.plan}
		// The framework's wrapper skips create and destroy by looking at the
		// raw state and plan, which a bare request leaves null; give it
		// something non-null so the function under test is reached.
		req.State.Raw = nonNullObject()
		req.Plan.Raw = nonNullObject()
		resp := &planmodifier.StringResponse{PlanValue: tc.plan}
		requiresReplaceIfRemoved().PlanModifyString(context.Background(), req, resp)
		if resp.RequiresReplace != tc.want {
			t.Errorf("%s: requires replace = %v, want %v", tc.name, resp.RequiresReplace, tc.want)
		}
	}
}

func nonNullObject() tftypes.Value {
	return tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})
}

type capturedRequest struct {
	method string
	path   string
	body   map[string]any
}

// targetServer answers target and user routes and records the requests. A
// response for a "METHOD path" key overrides the default target body.
func targetServer(t *testing.T, responses map[string]scripted) (*httptest.Server, func() []capturedRequest) {
	t.Helper()

	var mu sync.Mutex
	var requests []capturedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		requests = append(requests, capturedRequest{method: r.Method, path: r.URL.Path, body: body})
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if s, ok := responses[r.Method+" "+r.URL.Path]; ok {
			w.WriteHeader(s.status)
			_ = json.NewEncoder(w).Encode(s.body)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"$id": "target-1", "userId": "user-1", "providerType": "email", "identifier": "a@example.com",
			"name": "", "providerId": nil, "expired": false,
		})
	}))
	t.Cleanup(server.Close)
	return server, func() []capturedRequest {
		mu.Lock()
		defer mu.Unlock()
		return append([]capturedRequest(nil), requests...)
	}
}

type scripted struct {
	status int
	body   any
}

func appwriteError(status int, errorType string) scripted {
	return scripted{status: status, body: map[string]any{"message": "scripted", "code": status, "type": errorType}}
}

func testUsersClient(url string) *users.Users {
	return appwrite.NewUsers(appwrite.NewClient(appwrite.WithEndpoint(url+"/v1"), appwrite.WithProject("test")))
}

func TestCreateTarget(t *testing.T) {
	t.Parallel()

	plan := func(name, providerID types.String) targetResourceModel {
		return targetResourceModel{
			UserID:       types.StringValue("user-1"),
			ProviderType: types.StringValue("email"),
			Identifier:   types.StringValue("a@example.com"),
			Name:         name,
			ProviderID:   providerID,
		}
	}

	t.Run("without a provider it only creates", func(t *testing.T) {
		t.Parallel()
		server, requests := targetServer(t, nil)

		target, err := createTarget(testUsersClient(server.URL), plan(types.StringNull(), types.StringNull()), "target-1")
		if err != nil || target == nil {
			t.Fatalf("got %v, %v", target, err)
		}
		got := requests()
		if len(got) != 1 || got[0].method != http.MethodPost || got[0].path != "/v1/users/user-1/targets" {
			t.Fatalf("requests = %+v", got)
		}
		want := map[string]any{"targetId": "target-1", "providerType": "email", "identifier": "a@example.com"}
		if !equalBody(got[0].body, want) {
			t.Errorf("body = %v, want %v", got[0].body, want)
		}
	})

	t.Run("with a provider the server checks it on update", func(t *testing.T) {
		t.Parallel()
		server, requests := targetServer(t, nil)

		_, err := createTarget(testUsersClient(server.URL), plan(types.StringValue("Work"), types.StringValue("smtp-1")), "target-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := requests()
		if len(got) != 2 {
			t.Fatalf("requests = %+v", got)
		}
		if got[0].body["providerId"] != "smtp-1" || got[0].body["name"] != "Work" {
			t.Errorf("create body = %v", got[0].body)
		}
		if got[1].method != http.MethodPatch || got[1].path != "/v1/users/user-1/targets/target-1" {
			t.Errorf("follow-up = %s %s", got[1].method, got[1].path)
		}
		if !equalBody(got[1].body, map[string]any{"providerId": "smtp-1"}) {
			t.Errorf("follow-up body = %v", got[1].body)
		}
	})

	t.Run("a refused provider still returns the created target", func(t *testing.T) {
		t.Parallel()
		server, _ := targetServer(t, map[string]scripted{
			"PATCH /v1/users/user-1/targets/target-1": appwriteError(http.StatusBadRequest, "provider_incorrect_type"),
		})

		target, err := createTarget(testUsersClient(server.URL), plan(types.StringNull(), types.StringValue("sms-1")), "target-1")
		if err == nil {
			t.Fatal("expected an error")
		}
		if target == nil || target.Id != "target-1" {
			t.Fatalf("target = %+v", target)
		}
	})

	t.Run("a duplicate explains that identifiers are unique", func(t *testing.T) {
		t.Parallel()
		server, _ := targetServer(t, map[string]scripted{
			"POST /v1/users/user-1/targets": appwriteError(http.StatusConflict, "user_target_already_exists"),
		})

		target, err := createTarget(testUsersClient(server.URL), plan(types.StringNull(), types.StringNull()), "target-1")
		if err == nil || target != nil {
			t.Fatalf("got %v, %v", target, err)
		}
		if msg := formatTargetError(err, true); !strings.Contains(msg, "unique across the whole project") {
			t.Errorf("message does not explain the duplicate: %s", msg)
		}
	})
}

func TestTargetUpdateOptions(t *testing.T) {
	t.Parallel()

	state := targetResourceModel{
		UserID:     types.StringValue("user-1"),
		ID:         types.StringValue("target-1"),
		Identifier: types.StringValue("a@example.com"),
		ProviderID: types.StringValue("smtp-1"),
		Name:       types.StringValue("Work"),
	}

	for _, tc := range []struct {
		name string
		plan func(targetResourceModel) targetResourceModel
		want map[string]any
	}{
		{
			name: "only the name",
			plan: func(m targetResourceModel) targetResourceModel { m.Name = types.StringValue("Home"); return m },
			// The identifier is left out: sending it would clear expired.
			want: map[string]any{"name": "Home"},
		},
		{
			name: "only the identifier",
			plan: func(m targetResourceModel) targetResourceModel {
				m.Identifier = types.StringValue("b@example.com")
				return m
			},
			want: map[string]any{"identifier": "b@example.com"},
		},
		{
			name: "everything",
			plan: func(m targetResourceModel) targetResourceModel {
				m.Identifier = types.StringValue("b@example.com")
				m.ProviderID = types.StringValue("smtp-2")
				m.Name = types.StringValue("Home")
				return m
			},
			want: map[string]any{"identifier": "b@example.com", "providerId": "smtp-2", "name": "Home"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server, requests := targetServer(t, nil)
			client := testUsersClient(server.URL)

			if _, err := client.UpdateTarget("user-1", "target-1", targetUpdateOptions(client, state, tc.plan(state))...); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := requests()
			if len(got) != 1 || !equalBody(got[0].body, tc.want) {
				t.Errorf("requests = %+v, want body %v", got, tc.want)
			}
		})
	}
}

func TestCheckTargetImportable(t *testing.T) {
	t.Parallel()

	user := map[string]any{
		"$id":   "user-1",
		"email": "someone@example.com",
		"phone": "+14155550100",
		"targets": []map[string]any{
			{"$id": "auto-email", "userId": "user-1", "providerType": "email", "identifier": "someone@example.com"},
			{"$id": "auto-sms", "userId": "user-1", "providerType": "sms", "identifier": "+14155550100"},
			{"$id": "custom", "userId": "user-1", "providerType": "email", "identifier": "alerts@example.com"},
		},
	}
	server, _ := targetServer(t, map[string]scripted{
		"GET /v1/users/user-1":  {status: http.StatusOK, body: user},
		"GET /v1/users/missing": appwriteError(http.StatusNotFound, "user_not_found"),
	})
	client := testUsersClient(server.URL)

	for _, tc := range []struct {
		userID, targetID string
		wantErr          string
	}{
		{"user-1", "custom", ""},
		{"user-1", "auto-email", "own contact target"},
		{"user-1", "auto-sms", "own contact target"},
		{"user-1", "absent", "has no target"},
		{"missing", "custom", "Error reading user"},
	} {
		diags := checkTargetImportable(client, tc.userID, tc.targetID)
		switch {
		case tc.wantErr == "" && diags.HasError():
			t.Errorf("%s/%s: unexpected error %v", tc.userID, tc.targetID, diags)
		case tc.wantErr != "" && (!diags.HasError() || !strings.Contains(diags[0].Summary()+diags[0].Detail(), tc.wantErr)):
			t.Errorf("%s/%s: want error containing %q, got %v", tc.userID, tc.targetID, tc.wantErr, diags)
		}
	}
}

func equalBody(got, want map[string]any) bool {
	if len(got) != len(want) {
		return false
	}
	for k, v := range want {
		if got[k] != v {
			return false
		}
	}
	return true
}
