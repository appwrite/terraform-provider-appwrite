package project

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/models"
	sdkproject "github.com/appwrite/sdk-for-go/v7/project"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type capturedRequest struct {
	method string
	path   string
	body   map[string]any
}

// platformServer answers every request with a platform of the given type,
// echoing the name and identifier it was sent, and keeps the requests it
// received.
func platformServer(t *testing.T, platformType, apiField string) (*httptest.Server, *[]capturedRequest) {
	t.Helper()

	var requests []capturedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		requests = append(requests, capturedRequest{method: r.Method, path: r.URL.Path, body: body})

		// Appwrite answers a delete with an empty 204 and no content type.
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		response := map[string]any{
			"$id":        "platform-id",
			"$createdAt": "2026-11-04T00:00:00.000+00:00",
			"$updatedAt": "2026-11-04T00:00:00.000+00:00",
			"type":       platformType,
			"name":       "stored-name",
			apiField:     "stored-" + apiField,
		}
		if name, ok := body["name"]; ok {
			response["name"] = name
			response[apiField] = body[apiField]
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	t.Cleanup(server.Close)
	return server, &requests
}

func testProjectClient(url string) *sdkproject.Project {
	// No retries: the SDK client here is bare, so each scripted response is
	// seen by the code under test exactly once.
	return appwrite.NewProject(appwrite.NewClient(appwrite.WithEndpoint(url+"/v1"), appwrite.WithProject("project-id")))
}

// TestPlatformRoutes sends a create, update, get and delete for every
// platform type and checks the route, the arguments that reach the server and
// what lands in state.
//
// Each type has its own create and update route, and its own name for the
// identifying field, so a type wired to the wrong call or field would
// otherwise only show up against a server.
func TestPlatformRoutes(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		platformType string
		apiField     string
		set          func(*platformResourceModel, string)
	}{
		{"web", "hostname", func(m *platformResourceModel, v string) { m.Hostname = types.StringValue(v) }},
		{"android", "applicationId", func(m *platformResourceModel, v string) { m.ApplicationID = types.StringValue(v) }},
		{"apple", "bundleIdentifier", func(m *platformResourceModel, v string) { m.BundleIdentifier = types.StringValue(v) }},
		{"linux", "packageName", func(m *platformResourceModel, v string) { m.PackageName = types.StringValue(v) }},
		{"windows", "packageIdentifierName", func(m *platformResourceModel, v string) {
			m.PackageIdentifierName = types.StringValue(v)
		}},
	} {
		t.Run(tc.platformType, func(t *testing.T) {
			t.Parallel()
			server, requests := platformServer(t, tc.platformType, tc.apiField)
			client := testProjectClient(server.URL)

			plan := platformResourceModel{
				ID:   types.StringValue("platform-id"),
				Type: types.StringValue(tc.platformType),
				Name: types.StringValue("created"),
			}
			tc.set(&plan, "created-value")

			created, err := createPlatform(client, "platform-id", plan)
			if err != nil {
				t.Fatalf("create: %v", err)
			}
			plan.Name = types.StringValue("updated")
			tc.set(&plan, "updated-value")
			updated, err := updatePlatform(client, plan)
			if err != nil {
				t.Fatalf("update: %v", err)
			}
			read, err := client.GetPlatform("platform-id")
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			if _, err := client.DeletePlatform("platform-id"); err != nil {
				t.Fatalf("delete: %v", err)
			}

			for n, want := range []struct {
				method, path string
				body         map[string]any
			}{
				{http.MethodPost, "/v1/project/platforms/" + tc.platformType,
					map[string]any{"platformId": "platform-id", "name": "created", tc.apiField: "created-value"}},
				{http.MethodPut, "/v1/project/platforms/" + tc.platformType + "/platform-id",
					map[string]any{"name": "updated", tc.apiField: "updated-value"}},
				{http.MethodGet, "/v1/project/platforms/platform-id", nil},
				{http.MethodDelete, "/v1/project/platforms/platform-id", nil},
			} {
				got := (*requests)[n]
				if got.method != want.method || got.path != want.path {
					t.Errorf("request %d: got %s %s, want %s %s", n, got.method, got.path, want.method, want.path)
				}
				for key, value := range want.body {
					if got.body[key] != value {
						t.Errorf("request %d: %s = %v, want %v", n, key, got.body[key], value)
					}
				}
				if want.body != nil && len(got.body) != len(want.body) {
					t.Errorf("request %d: sent %v, want only %v", n, got.body, want.body)
				}
			}

			for step, platform := range map[string]models.Model{"created": created, "updated": updated, "read": read} {
				var model platformResourceModel
				if err := mapPlatformToState(platform, &model); err != nil {
					t.Fatalf("%s: %v", step, err)
				}
				if model.Type.ValueString() != tc.platformType || model.ID.ValueString() != "platform-id" {
					t.Errorf("%s: got type %s, id %s", step, model.Type, model.ID)
				}
				value := model.identifierField(tc.platformType).ValueString()
				want := map[string]string{"created": "created-value", "updated": "updated-value", "read": "stored-" + tc.apiField}[step]
				if value != want {
					t.Errorf("%s: identifier %q, want %q", step, value, want)
				}
			}
		})
	}
}

// TestPlatformErrorDetail checks that the two errors a user can act on carry
// guidance, and that a missing scope names the scopes to add.
func TestPlatformErrorDetail(t *testing.T) {
	t.Parallel()

	for errorType, want := range map[string]string{
		"platform_already_exists":     "unique across the whole Appwrite instance",
		"platform_method_unsupported": "Replace it instead",
		"general_unauthorized_scope":  "platforms.read and platforms.write",
	} {
		t.Run(errorType, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "message", "code": 400, "type": errorType})
			}))
			t.Cleanup(server.Close)

			_, err := testProjectClient(server.URL).CreateWebPlatform("platform-id", "name", "app.example.com")
			if err == nil {
				t.Fatal("expected an error")
			}
			if got := platformErrorDetail(err); !strings.Contains(got, want) {
				t.Errorf("%q does not mention %q", got, want)
			}
		})
	}
}
