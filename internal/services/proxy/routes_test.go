package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/proxy"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type capturedRequest struct {
	method string
	path   string
	body   map[string]any
}

// ruleServer answers every request with a rule and keeps the requests it
// received.
func ruleServer(t *testing.T) (*httptest.Server, *[]capturedRequest) {
	t.Helper()

	var requests []capturedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		requests = append(requests, capturedRequest{method: r.Method, path: r.URL.Path, body: body})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"$id": "rule-id"})
	}))
	t.Cleanup(server.Close)
	return server, &requests
}

func testProxyClient(url string) *proxy.Proxy {
	return appwrite.NewProxy(appwrite.NewClient(appwrite.WithEndpoint(url+"/v1"), appwrite.WithProject("project-id")))
}

// TestRuleRoutes creates a rule of every type and checks the route and the
// exact body that reach the server.
//
// Each type has its own route and SDK call. The acceptance tests run against
// a server that cannot verify the test domains, so this is what pins a type
// to the right call and each argument to the right parameter.
func TestRuleRoutes(t *testing.T) {
	t.Parallel()

	null := types.StringNull()
	s := types.StringValue

	for _, tc := range []struct {
		name  string
		plan  ruleResourceModel
		route string
		body  map[string]any
	}{
		{
			name:  "site",
			plan:  ruleResourceModel{Type: s("site"), ResourceID: s("site-id"), Branch: null},
			route: "/v1/proxy/rules/site",
			body:  map[string]any{"domain": "rule.example.com", "siteId": "site-id"},
		},
		{
			name:  "site with branch",
			plan:  ruleResourceModel{Type: s("site"), ResourceID: s("site-id"), Branch: s("main")},
			route: "/v1/proxy/rules/site",
			body:  map[string]any{"domain": "rule.example.com", "siteId": "site-id", "branch": "main"},
		},
		{
			name:  "function",
			plan:  ruleResourceModel{Type: s("function"), ResourceID: s("function-id"), Branch: null},
			route: "/v1/proxy/rules/function",
			body:  map[string]any{"domain": "rule.example.com", "functionId": "function-id"},
		},
		{
			name:  "function with branch",
			plan:  ruleResourceModel{Type: s("function"), ResourceID: s("function-id"), Branch: s("main")},
			route: "/v1/proxy/rules/function",
			body:  map[string]any{"domain": "rule.example.com", "functionId": "function-id", "branch": "main"},
		},
		{
			name:  "api",
			plan:  ruleResourceModel{Type: s("api"), ResourceID: null, Branch: null},
			route: "/v1/proxy/rules/api",
			body:  map[string]any{"domain": "rule.example.com"},
		},
		{
			name: "redirect",
			plan: ruleResourceModel{
				Type:                 s("redirect"),
				ResourceID:           s("site-id"),
				Branch:               null,
				RedirectURL:          s("https://example.org/landing"),
				RedirectStatusCode:   types.Int64Value(308),
				RedirectResourceType: s("site"),
			},
			route: "/v1/proxy/rules/redirect",
			body: map[string]any{
				"domain":       "rule.example.com",
				"url":          "https://example.org/landing",
				"statusCode":   "308",
				"resourceId":   "site-id",
				"resourceType": "site",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server, requests := ruleServer(t)

			plan := tc.plan
			plan.Domain = s("rule.example.com")
			if _, err := createRule(testProxyClient(server.URL), plan); err != nil {
				t.Fatalf("create: %v", err)
			}

			if len(*requests) != 1 {
				t.Fatalf("got %d requests, want 1", len(*requests))
			}
			got := (*requests)[0]
			if got.method != http.MethodPost || got.path != tc.route {
				t.Errorf("got %s %s, want POST %s", got.method, got.path, tc.route)
			}
			if !reflect.DeepEqual(got.body, tc.body) {
				t.Errorf("body = %v, want %v", got.body, tc.body)
			}
		})
	}
}

func TestCreateRuleRejectsUnknownType(t *testing.T) {
	t.Parallel()
	server, requests := ruleServer(t)

	_, err := createRule(testProxyClient(server.URL), ruleResourceModel{Type: types.StringValue("deployment")})
	if err == nil {
		t.Fatal("expected an error for an unsupported type")
	}
	if len(*requests) != 0 {
		t.Errorf("sent %d requests for an unsupported type", len(*requests))
	}
}
