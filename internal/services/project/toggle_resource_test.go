package project

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/models"
	sdkproject "github.com/appwrite/sdk-for-go/v7/project"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// project220 is the services and protocols fields of GET /v1/project on
// Appwrite 2.2.0 for a new project, trimmed to what these resources read.
const project220 = `{
  "$id": "project-id",
  "services": [
    {"$id": "account", "enabled": true}, {"$id": "avatars", "enabled": true},
    {"$id": "databases", "enabled": true}, {"$id": "tablesdb", "enabled": true},
    {"$id": "locale", "enabled": true}, {"$id": "health", "enabled": true},
    {"$id": "project", "enabled": true}, {"$id": "storage", "enabled": true},
    {"$id": "teams", "enabled": true}, {"$id": "users", "enabled": true},
    {"$id": "vcs", "enabled": true}, {"$id": "sites", "enabled": true},
    {"$id": "functions", "enabled": true}, {"$id": "proxy", "enabled": true},
    {"$id": "graphql", "enabled": true}, {"$id": "migrations", "enabled": true},
    {"$id": "messaging", "enabled": true}, {"$id": "advisor", "enabled": true}
  ],
  "protocols": [
    {"$id": "rest", "enabled": true}, {"$id": "graphql", "enabled": true},
    {"$id": "websocket", "enabled": true}
  ]
}`

// TestToggleItemsMatchServer pins the schema to what Appwrite 2.2.0 reports.
// A switch the server has and the schema lacks cannot be managed; one the
// schema has and the server lacks fails at apply with a 400.
func TestToggleItemsMatchServer(t *testing.T) {
	t.Parallel()

	var project models.Project
	if err := json.Unmarshal([]byte(project220), &project); err != nil {
		t.Fatal(err)
	}

	for _, kind := range []toggleKind{servicesKind, protocolsKind} {
		var want []string
		for id := range kind.current(&project) {
			want = append(want, id)
		}
		var got []string
		for _, item := range kind.items {
			got = append(got, item.id)
		}
		if !sameElements(got, want) {
			t.Errorf("%s: schema has %v, server reports %v", kind.typeSuffix, got, want)
		}
	}
}

func sameElements(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := map[string]int{}
	for _, s := range a {
		seen[s]++
	}
	for _, s := range b {
		seen[s]--
	}
	for _, n := range seen {
		if n != 0 {
			return false
		}
	}
	return true
}

func TestChangesFor(t *testing.T) {
	t.Parallel()

	items := []toggleItem{{id: "a"}, {id: "b"}, {id: "c"}, {id: "d"}}
	on, off, null := types.BoolValue(true), types.BoolValue(false), types.BoolNull()

	for _, tc := range []struct {
		name           string
		prior, planned toggleValues
		want           []toggleChange
	}{
		{
			name:    "create sends every set value, true included",
			prior:   toggleValues{},
			planned: toggleValues{"a": off, "b": on, "c": null},
			want:    []toggleChange{{"a", false}, {"b", true}},
		},
		{
			name:    "unchanged values are not sent",
			prior:   toggleValues{"a": off, "b": on},
			planned: toggleValues{"a": off, "b": on},
			want:    nil,
		},
		{
			name:    "a changed value is sent",
			prior:   toggleValues{"a": off, "b": on},
			planned: toggleValues{"a": on, "b": off},
			want:    []toggleChange{{"a", true}, {"b", false}},
		},
		{
			name:    "a value removed from configuration is enabled",
			prior:   toggleValues{"a": off, "b": on},
			planned: toggleValues{"a": null, "b": null},
			want:    []toggleChange{{"a", true}, {"b", true}},
		},
		{
			name:    "a value newly set is sent",
			prior:   toggleValues{"a": null},
			planned: toggleValues{"a": off, "d": on},
			want:    []toggleChange{{"a", false}, {"d", true}},
		},
		{
			name:    "destroy enables everything managed and nothing else",
			prior:   toggleValues{"a": off, "b": on, "c": null},
			planned: toggleValues{},
			want:    []toggleChange{{"a", true}, {"b", true}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := changesFor(items, tc.prior, tc.planned)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRefreshValues(t *testing.T) {
	t.Parallel()

	items := []toggleItem{{id: "managed"}, {id: "unmanaged"}, {id: "unknown-to-server"}}
	state := toggleValues{
		"managed":           types.BoolValue(true),
		"unmanaged":         types.BoolNull(),
		"unknown-to-server": types.BoolValue(false),
	}
	got := refreshValues(items, state, map[string]bool{"managed": false, "unmanaged": false})

	want := toggleValues{
		// Drift on a managed switch is reported.
		"managed": types.BoolValue(false),
		// A switch Terraform does not manage stays null whatever the server
		// says, so it never shows as a diff.
		"unmanaged": types.BoolNull(),
		// A switch the server does not report keeps its state value.
		"unknown-to-server": types.BoolValue(false),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

type toggleRequest struct {
	method  string
	path    string
	project string
	body    map[string]any
}

// toggleServer answers every request with a project and records it. A request
// whose path is failPath gets a 400 instead.
func toggleServer(t *testing.T, failPath string) (*sdkproject.Project, func() []toggleRequest) {
	t.Helper()

	var mu sync.Mutex
	var requests []toggleRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		requests = append(requests, toggleRequest{method: r.Method, path: r.URL.Path, project: r.Header.Get("X-Appwrite-Project"), body: body})
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == failPath {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message":"Invalid param","code":400,"type":"general_argument_invalid"}`))
			return
		}
		_, _ = w.Write([]byte(project220))
	}))
	t.Cleanup(server.Close)

	// No retries: the SDK client here is bare, so each response is seen by the
	// code under test exactly once.
	client := appwrite.NewProject(appwrite.NewClient(appwrite.WithEndpoint(server.URL+"/v1"), appwrite.WithProject("project-id")))
	return client, func() []toggleRequest {
		mu.Lock()
		defer mu.Unlock()
		return append([]toggleRequest(nil), requests...)
	}
}

// TestToggleRoutes disables and then enables every service and protocol and
// checks the route, the project header and the body that reach the server.
func TestToggleRoutes(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		kind  toggleKind
		route string
	}{
		{servicesKind, "/v1/project/services/"},
		{protocolsKind, "/v1/project/protocols/"},
	} {
		t.Run(tc.kind.noun, func(t *testing.T) {
			t.Parallel()
			client, requests := toggleServer(t, "")

			disabled, enabled := toggleValues{}, toggleValues{}
			for _, item := range tc.kind.items {
				disabled[item.id] = types.BoolValue(false)
			}
			// Disable everything, then remove every attribute from
			// configuration, which enables it all again.
			for _, step := range []struct{ prior, planned toggleValues }{{toggleValues{}, disabled}, {disabled, enabled}} {
				changes := changesFor(tc.kind.items, step.prior, step.planned)
				if applied, err := applyChanges(client, tc.kind, changes); err != nil || applied != len(changes) {
					t.Fatalf("applied %d of %d: %v", applied, len(changes), err)
				}
			}

			got := requests()
			if len(got) != 2*len(tc.kind.items) {
				t.Fatalf("got %d requests, want %d", len(got), 2*len(tc.kind.items))
			}
			for n, request := range got {
				item := tc.kind.items[n%len(tc.kind.items)]
				wantEnabled := n >= len(tc.kind.items)
				if request.method != http.MethodPatch || request.path != tc.route+item.id {
					t.Errorf("request %d: got %s %s, want PATCH %s%s", n, request.method, request.path, tc.route, item.id)
				}
				if request.project != "project-id" {
					t.Errorf("request %d: X-Appwrite-Project = %q", n, request.project)
				}
				if !reflect.DeepEqual(request.body, map[string]any{"enabled": wantEnabled}) {
					t.Errorf("request %d: body %v, want enabled=%v", n, request.body, wantEnabled)
				}
			}
		})
	}
}

// TestApplyChangesStopsAtFailure checks that a failed call stops the run and
// that the state written afterwards records only the calls that were made.
func TestApplyChangesStopsAtFailure(t *testing.T) {
	t.Parallel()

	client, requests := toggleServer(t, "/v1/project/protocols/graphql")

	off, null := types.BoolValue(false), types.BoolNull()
	prior := toggleValues{}
	planned := toggleValues{"rest": off, "graphql": off, "websocket": off}
	changes := changesFor(protocolsKind.items, prior, planned)

	applied, err := applyChanges(client, protocolsKind, changes)
	if err == nil {
		t.Fatal("expected an error")
	}
	if want := `disabling protocol "graphql": Invalid param`; err.Error() != want {
		t.Errorf("error %q, want %q", err.Error(), want)
	}
	if applied != 1 {
		t.Errorf("applied = %d, want 1", applied)
	}
	if n := len(requests()); n != 2 {
		t.Errorf("got %d requests, want 2: nothing is sent after a failure", n)
	}

	got := resultValues(protocolsKind.items, prior, planned, changes[applied:])
	want := toggleValues{"rest": off, "graphql": null, "websocket": null}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("state %v, want %v", got, want)
	}
}
