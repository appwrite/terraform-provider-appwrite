package organization

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/client"
	sdkorganization "github.com/appwrite/sdk-for-go/v7/organization"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

const (
	testOrganizationID = "org-id"
	testKey            = "organization_test-secret"
)

type capturedRequest struct {
	method       string
	path         string
	project      string
	organization string
	key          string
	body         map[string]any
}

// membershipServer answers the organization membership routes and keeps the
// requests it received. A create returns createRoles when set, standing in
// for a pending invitation the server confirms with the roles it was sent
// with.
type membershipServer struct {
	mu          sync.Mutex
	requests    []capturedRequest
	createRoles []string
	// errorType, when set, is returned as a 400 from every route.
	errorType string
}

func newMembershipServer(t *testing.T) (*membershipServer, *httptest.Server) {
	t.Helper()
	fake := &membershipServer{}
	server := httptest.NewServer(http.HandlerFunc(fake.serve))
	t.Cleanup(server.Close)
	return fake, server
}

func (f *membershipServer) serve(w http.ResponseWriter, r *http.Request) {
	body := map[string]any{}
	_ = json.NewDecoder(r.Body).Decode(&body)
	// GET and DELETE carry their parameters in the query string.
	for key, values := range r.URL.Query() {
		body[key] = values[0]
	}

	f.mu.Lock()
	f.requests = append(f.requests, capturedRequest{
		method:       r.Method,
		path:         r.URL.Path,
		project:      r.Header.Get("X-Appwrite-Project"),
		organization: r.Header.Get("X-Appwrite-Organization"),
		key:          r.Header.Get("X-Appwrite-Key"),
		body:         body,
	})
	createRoles, errorType := f.createRoles, f.errorType
	f.mu.Unlock()

	if errorType != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "refused", "code": 400, "type": errorType})
		return
	}
	// Appwrite answers a delete with an empty 204 and no JSON content type.
	if r.Method == http.MethodDelete {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	roles := []any{"developer"}
	if requested, ok := body["roles"].([]any); ok {
		roles = requested
	}
	if r.Method == http.MethodPost && createRoles != nil {
		roles = nil
		for _, role := range createRoles {
			roles = append(roles, role)
		}
	}
	if r.Method == http.MethodPost {
		w.WriteHeader(http.StatusCreated)
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"$id":       "membership-id",
		"userId":    "user-id",
		"userName":  "Jane",
		"userEmail": "jane@example.com",
		"teamId":    testOrganizationID,
		"invited":   "2026-11-04T10:00:00.000+00:00",
		"joined":    "2026-11-04T10:00:00.000+00:00",
		"confirm":   true,
		"roles":     roles,
	})
}

func (f *membershipServer) captured() []capturedRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]capturedRequest(nil), f.requests...)
}

// testClient authenticates the way the provider does: the organization key,
// scoped to the organization through ClientForOrganization.
func testClient(t *testing.T, url string) *sdkorganization.Organization {
	t.Helper()
	c, err := membershipClient(testClients(url), testOrganizationID)
	if err != nil {
		t.Fatalf("membershipClient: %v", err)
	}
	return c
}

func testClients(url string) *common.AppwriteClients {
	return &common.AppwriteClients{
		// No retries: each scripted response is seen exactly once.
		OrganizationBaseOptions: []client.ClientOption{
			appwrite.WithEndpoint(url + "/v1"),
			appwrite.WithKey(testKey),
		},
		OrganizationCredentialType: common.CredentialTypeOrganization,
		OrganizationID:             testOrganizationID,
	}
}

// TestMembershipRoutes sends every call the resource makes to a local server
// and checks the route, the headers that scope it to the organization, and
// the arguments.
//
// The acceptance tests need Appwrite Cloud and an account set aside for
// testing, so they do not run in CI. This is the coverage that does.
func TestMembershipRoutes(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		plan membershipResourceModel
		want map[string]any
	}{
		"by email": {
			plan: membershipResourceModel{
				Email: types.StringValue("Jane@Example.com"),
				Name:  types.StringValue("Jane"),
			},
			want: map[string]any{"email": "Jane@Example.com", "name": "Jane"},
		},
		"by user ID": {
			plan: membershipResourceModel{
				// Unknown until apply, as it is when only email is configured.
				Email:  types.StringUnknown(),
				UserID: types.StringValue("user-id"),
				Name:   types.StringNull(),
			},
			want: map[string]any{"userId": "user-id"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fake, server := newMembershipServer(t)
			c := testClient(t, server.URL)

			if _, err := createMembership(c, tc.plan, []string{"developer", "editor"}); err != nil {
				t.Fatalf("create: %v", err)
			}
			if _, err := updateMembership(c, "membership-id", []string{"analyst"}, true); err != nil {
				t.Fatalf("update: %v", err)
			}
			if _, err := updateMembership(c, "membership-id", []string{"analyst"}, false); err != nil {
				t.Fatalf("update without role change: %v", err)
			}
			if _, err := c.GetMembership("membership-id"); err != nil {
				t.Fatalf("get: %v", err)
			}
			if _, err := c.DeleteMembership("membership-id"); err != nil {
				t.Fatalf("delete: %v", err)
			}

			requests := fake.captured()
			wantRoutes := []string{
				"POST /v1/organization/memberships",
				"PATCH /v1/organization/memberships/membership-id",
				// A change to email case or name has nothing to send.
				"GET /v1/organization/memberships/membership-id",
				"GET /v1/organization/memberships/membership-id",
				"DELETE /v1/organization/memberships/membership-id",
			}
			gotRoutes := make([]string, 0, len(requests))
			for _, req := range requests {
				gotRoutes = append(gotRoutes, req.method+" "+req.path)
				if req.project != "console" || req.organization != testOrganizationID || req.key != testKey {
					t.Errorf("%s %s: project %q, organization %q, key %q; want console, %s and the organization key",
						req.method, req.path, req.project, req.organization, req.key, testOrganizationID)
				}
			}
			if !reflect.DeepEqual(gotRoutes, wantRoutes) {
				t.Fatalf("routes:\n got %v\nwant %v", gotRoutes, wantRoutes)
			}

			create := requests[0].body
			if roles := create["roles"]; !reflect.DeepEqual(roles, []any{"developer", "editor"}) {
				t.Errorf("create roles = %v", roles)
			}
			for key, value := range tc.want {
				if create[key] != value {
					t.Errorf("create %s = %v, want %v", key, create[key], value)
				}
			}
			// An API key needs no redirect URL, and nothing else may be sent:
			// an empty email or user ID is still an argument to the server.
			for _, key := range []string{"email", "userId", "name", "phone", "url"} {
				if _, expected := tc.want[key]; !expected {
					if _, sent := create[key]; sent {
						t.Errorf("create sent %s = %v", key, create[key])
					}
				}
			}

			if roles := requests[1].body["roles"]; !reflect.DeepEqual(roles, []any{"analyst"}) {
				t.Errorf("update roles = %v", roles)
			}
		})
	}
}

// Appwrite reuses a membership the account already has. A pending invitation
// sent from the Console is confirmed in place and keeps the roles it was sent
// with, so the requested roles have to follow in an update.
func TestCreateMembershipAppliesRolesToAPendingInvitation(t *testing.T) {
	t.Parallel()
	fake, server := newMembershipServer(t)
	fake.createRoles = []string{"billing"}
	c := testClient(t, server.URL)

	plan := membershipResourceModel{Email: types.StringValue("jane@example.com")}
	got, err := createMembership(c, plan, []string{"developer"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if !reflect.DeepEqual(got.Roles, []string{"developer"}) {
		t.Errorf("roles = %v, want [developer]", got.Roles)
	}

	requests := fake.captured()
	if len(requests) != 2 || requests[1].method != http.MethodPatch {
		t.Fatalf("requests = %v, want a create followed by an update", requests)
	}
	if roles := requests[1].body["roles"]; !reflect.DeepEqual(roles, []any{"developer"}) {
		t.Errorf("update roles = %v", roles)
	}
}

func TestCreateMembershipSkipsUpdateWhenRolesMatch(t *testing.T) {
	t.Parallel()
	fake, server := newMembershipServer(t)
	// Same roles, in another order.
	fake.createRoles = []string{"editor", "developer"}
	c := testClient(t, server.URL)

	if _, err := createMembership(c, membershipResourceModel{Email: types.StringValue("jane@example.com")}, []string{"developer", "editor"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if n := len(fake.captured()); n != 1 {
		t.Errorf("sent %d requests, want only the create", n)
	}
}

func TestMembershipClientRejectsProjectKeys(t *testing.T) {
	t.Parallel()
	clients := testClients("http://127.0.0.1")
	clients.OrganizationCredentialType = common.CredentialTypeStandard

	_, err := membershipClient(clients, testOrganizationID)
	if err == nil {
		t.Fatal("a standard project key was accepted for organization memberships")
	}
	for _, want := range []string{"organization API key", "organization.memberships.read", "organization.memberships.write"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

// The server's refusals name the rule but not what to do about it.
func TestMembershipErrorExplainsRefusals(t *testing.T) {
	t.Parallel()

	for errorType, want := range map[string]string{
		"membership_deletion_prohibited":  "last confirmed owner",
		"membership_downgrade_prohibited": "at least one confirmed owner",
		"membership_already_confirmed":    "terraform import appwrite_organization_membership.",
	} {
		t.Run(errorType, func(t *testing.T) {
			t.Parallel()
			fake, server := newMembershipServer(t)
			fake.errorType = errorType

			_, err := testClient(t, server.URL).DeleteMembership("membership-id")
			if err == nil {
				t.Fatal("expected an error")
			}
			if got := membershipError(err); !strings.Contains(got, want) {
				t.Errorf("message %q does not contain %q", got, want)
			}
		})
	}
}

func TestMapMembershipToState(t *testing.T) {
	t.Parallel()
	_, server := newMembershipServer(t)
	membership, err := testClient(t, server.URL).GetMembership("membership-id")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	for name, tc := range map[string]struct {
		email     types.String
		wantEmail string
	}{
		// After an import, or when the member was added by user_id.
		"no email":    {email: types.StringNull(), wantEmail: "jane@example.com"},
		"unknown":     {email: types.StringUnknown(), wantEmail: "jane@example.com"},
		"other case":  {email: types.StringValue("Jane@Example.com"), wantEmail: "Jane@Example.com"},
		"old address": {email: types.StringValue("jane@old.example.com"), wantEmail: "jane@old.example.com"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			model := membershipResourceModel{Email: tc.email}
			if diags := mapMembershipToState(context.Background(), membership, &model); diags.HasError() {
				t.Fatalf("map: %v", diags)
			}
			if got := model.Email.ValueString(); got != tc.wantEmail {
				t.Errorf("email = %q, want %q", got, tc.wantEmail)
			}
			if got := model.UserEmail.ValueString(); got != "jane@example.com" {
				t.Errorf("user_email = %q, want the account's address", got)
			}
			if model.ID.ValueString() != "membership-id" || model.UserID.ValueString() != "user-id" ||
				model.UserName.ValueString() != "Jane" || !model.Confirm.ValueBool() ||
				model.JoinedAt.ValueString() != "2026-11-04T10:00:00.000+00:00" {
				t.Errorf("model = %+v", model)
			}
			want, _ := types.SetValueFrom(context.Background(), types.StringType, []string{"developer"})
			if !model.Roles.Equal(want) {
				t.Errorf("roles = %v", model.Roles)
			}
		})
	}
}
