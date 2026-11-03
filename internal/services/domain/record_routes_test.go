package domain

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

type capturedRequest struct {
	method string
	path   string
	body   map[string]any
}

// recordServer answers every record create and update with a record and
// keeps the requests it received.
func recordServer(t *testing.T) (*httptest.Server, *[]capturedRequest) {
	t.Helper()

	var requests []capturedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		requests = append(requests, capturedRequest{method: r.Method, path: r.URL.Path, body: body})

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"$id": "record-id", "type": "A"})
	}))
	t.Cleanup(server.Close)
	return server, &requests
}

// TestRecordRoutes sends a create and an update for every record type and
// checks the route and the arguments that reach the server.
//
// Each type has its own route and its own SDK call, so a type wired to the
// wrong call, or an argument passed in the wrong position, would otherwise
// only show up against Appwrite Cloud. The acceptance tests that reach Cloud
// are opt-in, so this is the coverage that runs in CI.
func TestRecordRoutes(t *testing.T) {
	t.Parallel()

	i := types.Int64Value
	null := types.Int64Null()

	for _, tc := range []struct {
		recordType string
		route      string
		priority   types.Int64
		weight     types.Int64
		port       types.Int64
		// want lists the numeric arguments the route must receive, beyond
		// the ttl every route takes.
		want map[string]float64
	}{
		{recordType: "A", route: "a", priority: null, weight: null, port: null},
		{recordType: "AAAA", route: "aaaa", priority: null, weight: null, port: null},
		{recordType: "ALIAS", route: "alias", priority: null, weight: null, port: null},
		{recordType: "CAA", route: "caa", priority: null, weight: null, port: null},
		{recordType: "CNAME", route: "cname", priority: null, weight: null, port: null},
		{recordType: "HTTPS", route: "https", priority: null, weight: null, port: null},
		{recordType: "MX", route: "mx", priority: i(10), weight: null, port: null, want: map[string]float64{"priority": 10}},
		{recordType: "NS", route: "ns", priority: null, weight: null, port: null},
		{recordType: "SRV", route: "srv", priority: i(10), weight: i(5), port: i(5060), want: map[string]float64{"priority": 10, "weight": 5, "port": 5060}},
		{recordType: "TXT", route: "txt", priority: null, weight: null, port: null},
	} {
		t.Run(tc.recordType, func(t *testing.T) {
			t.Parallel()
			server, requests := recordServer(t)
			client := testDomainsClient(server.URL)

			plan := recordResourceModel{
				ID:       types.StringValue("record-id"),
				DomainID: types.StringValue("domain-id"),
				Type:     types.StringValue(tc.recordType),
				Name:     types.StringValue("name-" + tc.route),
				Value:    types.StringValue("value-" + tc.route),
				TTL:      i(300),
				Priority: tc.priority,
				Weight:   tc.weight,
				Port:     tc.port,
				Comment:  types.StringValue("comment-" + tc.route),
			}

			if _, err := createRecord(client, plan); err != nil {
				t.Fatalf("create: %v", err)
			}
			// An update with the comment removed still sends it, empty, which
			// is what clears it on the server.
			plan.Comment = types.StringNull()
			if _, err := updateRecord(client, plan); err != nil {
				t.Fatalf("update: %v", err)
			}

			if len(*requests) != 2 {
				t.Fatalf("got %d requests, want 2", len(*requests))
			}
			for n, want := range []struct {
				method, path, comment string
			}{
				{http.MethodPost, "/v1/domains/domain-id/records/" + tc.route, "comment-" + tc.route},
				{http.MethodPut, "/v1/domains/domain-id/records/" + tc.route + "/record-id", ""},
			} {
				got := (*requests)[n]
				if got.method != want.method || got.path != want.path {
					t.Errorf("request %d: got %s %s, want %s %s", n, got.method, got.path, want.method, want.path)
				}
				checkBody(t, got.body, map[string]any{
					"name":    "name-" + tc.route,
					"value":   "value-" + tc.route,
					"ttl":     float64(300),
					"comment": want.comment,
				})
				for key, value := range tc.want {
					if got.body[key] != value {
						t.Errorf("request %d: %s = %v, want %v", n, key, got.body[key], value)
					}
				}
				for _, key := range []string{"priority", "weight", "port"} {
					if _, expected := tc.want[key]; !expected {
						if _, sent := got.body[key]; sent {
							t.Errorf("request %d: %s sent to the %s route, which does not take it", n, key, tc.route)
						}
					}
				}
			}
		})
	}
}

func checkBody(t *testing.T, body, want map[string]any) {
	t.Helper()
	var wrong []string
	for key, value := range want {
		if body[key] != value {
			wrong = append(wrong, key)
		}
	}
	if len(wrong) > 0 {
		t.Errorf("body %v: wrong %s", body, strings.Join(wrong, ", "))
	}
}
