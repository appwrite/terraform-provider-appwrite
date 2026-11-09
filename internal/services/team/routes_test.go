package team

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/client"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/sdk-for-go/v7/teams"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type capturedRequest struct {
	method string
	path   string
	query  string
	body   string
}

// scriptedServer answers each request with the response registered for its
// method and path, and keeps the requests it received.
func scriptedServer(t *testing.T, responses map[string]scripted) (*teams.Teams, *[]capturedRequest) {
	t.Helper()

	var requests []capturedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requests = append(requests, capturedRequest{method: r.Method, path: r.URL.Path, query: r.URL.RawQuery, body: string(body)})

		response, ok := responses[r.Method+" "+r.URL.Path]
		if !ok {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotImplemented)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if response.status != 0 {
			w.WriteHeader(response.status)
		}
		_, _ = io.WriteString(w, response.body)
	}))
	t.Cleanup(server.Close)

	// No retries: the SDK client here is bare, so each scripted response is
	// seen by the code under test exactly once.
	return appwrite.NewTeams(appwrite.NewClient(appwrite.WithEndpoint(server.URL+"/v1"), appwrite.WithProject("project"))), &requests
}

type scripted struct {
	status int
	body   string
}

const membershipJSON = `{"$id":"membership-id","teamId":"team-id","userId":"user-id","roles":["dev"],"confirm":true}`

// TestCreateMembershipBody checks what reaches the server for each way of
// naming the member. Only the identifier configured is sent, url never is,
// and an empty role set is sent as [] rather than null, which the server
// rejects.
func TestCreateMembershipBody(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		plan  membershipResourceModel
		roles []string
		want  map[string]any
	}{
		{
			name:  "user_id",
			plan:  membershipResourceModel{UserID: types.StringValue("user-id")},
			roles: []string{"dev"},
			want:  map[string]any{"userId": "user-id", "roles": []any{"dev"}},
		},
		{
			name:  "email and name",
			plan:  membershipResourceModel{UserID: types.StringUnknown(), Email: types.StringValue("A@example.com"), Name: types.StringValue("A")},
			roles: []string{"dev", "lead"},
			want:  map[string]any{"email": "A@example.com", "name": "A", "roles": []any{"dev", "lead"}},
		},
		{
			name:  "phone with no roles",
			plan:  membershipResourceModel{UserID: types.StringUnknown(), Phone: types.StringValue("+15550000000")},
			roles: nil,
			want:  map[string]any{"phone": "+15550000000", "roles": []any{}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// The response carries the roles sent, as for a new membership.
			created, _ := json.Marshal(map[string]any{"$id": "membership-id", "teamId": "team-id", "userId": "user-id", "roles": append([]string{}, tc.roles...), "confirm": true})
			client, requests := scriptedServer(t, map[string]scripted{
				"POST /v1/teams/team-id/memberships": {status: http.StatusCreated, body: string(created)},
			})
			// Fields left out are the zero types.String, which is null.
			tc.plan.TeamID = types.StringValue("team-id")

			if _, err := createMembership(client, tc.plan, tc.roles); err != nil {
				t.Fatal(err)
			}

			if len(*requests) != 1 {
				t.Fatalf("got %d requests, want 1", len(*requests))
			}
			var body map[string]any
			if err := json.Unmarshal([]byte((*requests)[0].body), &body); err != nil {
				t.Fatal(err)
			}
			if len(body) != len(tc.want) {
				t.Errorf("body %v, want exactly %v", body, tc.want)
			}
			for key, want := range tc.want {
				got, _ := json.Marshal(body[key])
				wantJSON, _ := json.Marshal(want)
				if string(got) != string(wantJSON) {
					t.Errorf("%s = %s, want %s", key, got, wantJSON)
				}
			}
			if _, sent := body["url"]; sent {
				t.Error("url sent; the server only uses it for invitation emails, which an API key never triggers")
			}
		})
	}
}

// TestCreateMembershipSetsConfiguredRoles covers a create that lands on a
// pending invitation for the same user: the server confirms and returns it with
// the roles it was sent with, so the configured roles are set afterwards. When
// the roles already match, nothing more is sent.
func TestCreateMembershipSetsConfiguredRoles(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		roles     []string
		wantPatch bool
	}{
		{name: "invitation roles differ", roles: []string{"lead"}, wantPatch: true},
		{name: "invitation roles match, repeats ignored", roles: []string{"dev", "dev"}, wantPatch: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client, requests := scriptedServer(t, map[string]scripted{
				"POST /v1/teams/team-id/memberships":                {status: http.StatusCreated, body: membershipJSON},
				"PATCH /v1/teams/team-id/memberships/membership-id": {body: `{"$id":"membership-id","teamId":"team-id","userId":"user-id","roles":["lead"],"confirm":true}`},
			})
			plan := membershipResourceModel{TeamID: types.StringValue("team-id"), UserID: types.StringValue("user-id")}

			membership, err := createMembership(client, plan, tc.roles)
			if err != nil {
				t.Fatal(err)
			}

			patched := len(*requests) == 2 && (*requests)[1].method == http.MethodPatch
			if patched != tc.wantPatch {
				t.Fatalf("requests %+v, want a roles update: %v", *requests, tc.wantPatch)
			}
			if !sameRoles(membership.Roles, tc.roles) {
				t.Errorf("roles = %v, want %v", membership.Roles, tc.roles)
			}
			if patched {
				var body map[string]any
				if err := json.Unmarshal([]byte((*requests)[1].body), &body); err != nil {
					t.Fatal(err)
				}
				if got, _ := json.Marshal(body["roles"]); string(got) != `["lead"]` {
					t.Errorf("update sent roles %s, want [\"lead\"]", got)
				}
			}
		})
	}
}

// TestAlreadyMemberHint checks the guidance added to the server's refusal of a
// membership that already exists.
func TestAlreadyMemberHint(t *testing.T) {
	t.Parallel()

	// Produce the error the way the SDK does, from a real 409.
	c, _ := scriptedServer(t, map[string]scripted{
		"POST /v1/teams/team-id/memberships": {status: http.StatusConflict, body: `{"message":"Membership is already confirmed.","code":409,"type":"membership_already_confirmed"}`},
	})
	_, conflict := c.CreateMembership("team-id", []string{})
	var appErr *client.AppwriteError
	if !errors.As(conflict, &appErr) {
		t.Fatalf("got %T %v, want *client.AppwriteError", conflict, conflict)
	}

	t.Run("finds the membership by user", func(t *testing.T) {
		t.Parallel()
		c, requests := scriptedServer(t, map[string]scripted{
			"GET /v1/teams/team-id/memberships": {body: `{"total":1,"memberships":[` + membershipJSON + `]}`},
		})
		hint := alreadyMemberHint(c, membershipResourceModel{TeamID: types.StringValue("team-id"), UserID: types.StringValue("user-id")}, conflict)
		if !strings.Contains(hint, "import it with the ID team-id/membership-id") {
			t.Errorf("hint %q does not name the import ID", hint)
		}
		if q := (*requests)[0].query; !strings.Contains(q, "userId") || !strings.Contains(q, "user-id") {
			t.Errorf("list query %q does not filter by the user", q)
		}
	})

	t.Run("by email names the format", func(t *testing.T) {
		t.Parallel()
		c, requests := scriptedServer(t, nil)
		hint := alreadyMemberHint(c, membershipResourceModel{TeamID: types.StringValue("team-id"), UserID: types.StringUnknown(), Email: types.StringValue("a@example.com")}, conflict)
		if !strings.Contains(hint, "team_id/membership_id") {
			t.Errorf("hint %q does not give the import format", hint)
		}
		if len(*requests) != 0 {
			t.Errorf("made %d requests; there is no user ID to search by", len(*requests))
		}
	})

	t.Run("other errors get none", func(t *testing.T) {
		t.Parallel()
		c, _ := scriptedServer(t, map[string]scripted{
			"POST /v1/teams/team-id/memberships": {status: http.StatusNotFound, body: `{"message":"User not found","code":404,"type":"user_not_found"}`},
		})
		_, err := c.CreateMembership("team-id", []string{})
		if hint := alreadyMemberHint(c, membershipResourceModel{TeamID: types.StringValue("team-id")}, err); hint != "" {
			t.Errorf("got hint %q for an unrelated error", hint)
		}
	})
}

// TestUpdatePrefsBody checks preferences reach the server as an object, with
// integers beyond 2^53 intact, and that the team is read back afterwards for
// its new timestamp.
func TestUpdatePrefsBody(t *testing.T) {
	t.Parallel()

	client, requests := scriptedServer(t, map[string]scripted{
		"PUT /v1/teams/team-id/prefs": {body: `{"big":9007199254740993}`},
		"GET /v1/teams/team-id":       {body: `{"$id":"team-id","name":"T","$updatedAt":"later","prefs":{"big":9007199254740993}}`},
	})

	team, err := updatePrefs(client, "team-id", `{"big":9007199254740993,"nested":{"a":[1,"b"]}}`)
	if err != nil {
		t.Fatal(err)
	}
	if team.UpdatedAt != "later" {
		t.Errorf("team not read back after the update: %+v", team)
	}
	if got, want := (*requests)[0].body, `{"prefs":{"big":9007199254740993,"nested":{"a":[1,"b"]}}}`; got != want {
		t.Errorf("body %s, want %s", got, want)
	}
}

// TestMapTeamToStateKeepsSpelling checks a refresh keeps the configured
// spelling of unchanged preferences.
func TestMapTeamToStateKeepsSpelling(t *testing.T) {
	t.Parallel()

	client, _ := scriptedServer(t, map[string]scripted{
		"GET /v1/teams/team-id": {body: `{"$id":"team-id","name":"T","prefs":{"theme":"dark","seats":1}}`},
	})
	team, err := client.Get("team-id")
	if err != nil {
		t.Fatal(err)
	}

	model := teamResourceModel{Prefs: types.StringValue(`{"seats":1.0,"theme":"dark"}`)}
	if diags := mapTeamToState(team, &model); diags.HasError() {
		t.Fatal(diags)
	}
	if got := model.Prefs.ValueString(); got != `{"seats":1.0,"theme":"dark"}` {
		t.Errorf("prefs = %s; the configured spelling should have been kept", got)
	}

	model = teamResourceModel{Prefs: types.StringNull()}
	if diags := mapTeamToState(team, &model); diags.HasError() {
		t.Fatal(diags)
	}
	if got := model.Prefs.ValueString(); got != `{"seats":1,"theme":"dark"}` {
		t.Errorf("prefs = %s, want the canonical form", got)
	}
}

func TestAdoptionMismatch(t *testing.T) {
	t.Parallel()

	membership := &models.Membership{Id: "m", UserId: "u", UserEmail: "a@example.com", UserPhone: "+15550000000"}
	imported := membershipResourceModel{Email: types.StringNull(), Phone: types.StringNull()}

	for _, tc := range []struct {
		name     string
		current  membershipResourceModel
		plan     membershipResourceModel
		mismatch bool
	}{
		{name: "matching email in another case", current: imported, plan: membershipResourceModel{Email: types.StringValue("A@Example.com"), Phone: types.StringNull()}},
		{name: "other email", current: imported, plan: membershipResourceModel{Email: types.StringValue("b@example.com"), Phone: types.StringNull()}, mismatch: true},
		{name: "matching phone", current: imported, plan: membershipResourceModel{Email: types.StringNull(), Phone: types.StringValue("+15550000000")}},
		{name: "other phone", current: imported, plan: membershipResourceModel{Email: types.StringNull(), Phone: types.StringValue("+15550000001")}, mismatch: true},
		// Only a value being set for the first time is checked; a change to
		// one already in state is a replacement, planned elsewhere.
		{name: "already set", current: membershipResourceModel{Email: types.StringValue("x@example.com"), Phone: types.StringNull()}, plan: membershipResourceModel{Email: types.StringValue("x@example.com"), Phone: types.StringNull()}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			msg := adoptionMismatch(tc.plan, tc.current, membership)
			if (msg != "") != tc.mismatch {
				t.Errorf("mismatch = %q, want mismatch %v", msg, tc.mismatch)
			}
		})
	}
}
