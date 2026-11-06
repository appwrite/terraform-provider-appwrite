package waf

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
)

type capturedRequest struct {
	method  string
	path    string
	project string
	body    map[string]any
}

// ruleServer answers every request with a minimal rule and keeps the
// requests it received. Only the requests matter here.
func ruleServer(t *testing.T) (*httptest.Server, *[]capturedRequest) {
	t.Helper()

	var requests []capturedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		requests = append(requests, capturedRequest{
			method:  r.Method,
			path:    r.URL.Path,
			project: r.Header.Get("X-Appwrite-Project"),
			body:    body,
		})

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"$id": "rule-id", "action": "deny", "conditions": []any{}, "config": map[string]any{}})
	}))
	t.Cleanup(server.Close)
	return server, &requests
}

func intPointer(v int) *int { return &v }

// TestRuleRoutes sends a create and an update for every rule type and checks
// the route and the exact arguments that reach the server.
//
// Each type has its own route and its own SDK call, so a type wired to the
// wrong call, or an argument sent to a route that does not take it, would
// otherwise only show up against Appwrite Cloud. The acceptance tests that
// reach Cloud are opt-in, so this is the coverage that runs in CI.
func TestRuleRoutes(t *testing.T) {
	t.Parallel()

	conditions := []string{`{"method":"equal","attribute":"ip","values":["198.51.100.10"]}`}

	for _, tc := range []struct {
		name  string
		route string
		spec  ruleSpec
		// createExtra and updateExtra are the type-specific arguments each
		// request must carry, beyond the ones every type takes.
		createExtra map[string]any
		updateExtra map[string]any
	}{
		{name: "bypass", route: "bypass", spec: ruleSpec{Type: typeBypass}},
		{name: "deny", route: "deny", spec: ruleSpec{Type: typeDeny}},
		{
			name: "challenge", route: "challenge",
			spec:        ruleSpec{Type: typeChallenge, ChallengeType: "compute", Difficulty: 4, TTL: 3600},
			createExtra: map[string]any{"challengeType": "compute", "difficulty": float64(4), "ttl": float64(3600)},
			updateExtra: map[string]any{"challengeType": "compute", "difficulty": float64(4), "ttl": float64(3600)},
		},
		{
			name: "rateLimit", route: "rate-limit",
			spec:        ruleSpec{Type: typeRateLimit, Limit: 100, Interval: 60, Key: "userId", Strategy: strategyFixedWindow},
			createExtra: map[string]any{"limit": float64(100), "interval": float64(60), "key": "userId", "strategy": "fixedWindow"},
			// The strategy cannot be changed after creation, so the update
			// route does not take it.
			updateExtra: map[string]any{"limit": float64(100), "interval": float64(60), "key": "userId"},
		},
		{
			name: "rateLimit tokenBucket", route: "rate-limit",
			spec: ruleSpec{Type: typeRateLimit, Limit: 100, Interval: 60, Key: "ip", Strategy: strategyTokenBucket,
				MaxBucketSize: intPointer(250)},
			createExtra: map[string]any{"limit": float64(100), "interval": float64(60), "key": "ip",
				"strategy": "tokenBucket", "maxBucketSize": float64(250)},
			updateExtra: map[string]any{"limit": float64(100), "interval": float64(60), "key": "ip",
				"maxBucketSize": float64(250)},
		},
		{
			name: "redirect", route: "redirect",
			spec:        ruleSpec{Type: typeRedirect, Location: "/maintenance", StatusCode: 307},
			createExtra: map[string]any{"location": "/maintenance", "statusCode": float64(307)},
			updateExtra: map[string]any{"location": "/maintenance", "statusCode": float64(307)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server, requests := ruleServer(t)
			client := testWafClient(server.URL)

			spec := tc.spec
			spec.ID = "rule-id"
			spec.Name = "name-" + tc.route
			spec.ResourceType = resourceTypeAPI
			spec.Priority = -5
			spec.Enabled = false
			spec.Conditions = conditions

			if _, err := createRule(client, spec); err != nil {
				t.Fatalf("create: %v", err)
			}
			if _, err := updateRule(client, spec); err != nil {
				t.Fatalf("update: %v", err)
			}
			if len(*requests) != 2 {
				t.Fatalf("got %d requests, want 2", len(*requests))
			}

			common := map[string]any{
				"name":         "name-" + tc.route,
				"resourceType": "api",
				"description":  "",
				"priority":     float64(-5),
				"enabled":      false,
				"conditions":   []any{conditions[0]},
			}
			createWant := merge(common, map[string]any{"ruleId": "rule-id"}, tc.createExtra)
			// An update sends the resource ID with the type every time,
			// because the API clears it whenever a type arrives without one.
			updateWant := merge(common, map[string]any{"resourceId": ""}, tc.updateExtra)

			for n, want := range []struct {
				method, path string
				body         map[string]any
			}{
				{http.MethodPost, "/v1/waf/rules/" + tc.route, createWant},
				{http.MethodPatch, "/v1/waf/rules/" + tc.route + "/rule-id", updateWant},
			} {
				got := (*requests)[n]
				if got.method != want.method || got.path != want.path {
					t.Errorf("request %d: got %s %s, want %s %s", n, got.method, got.path, want.method, want.path)
				}
				if got.project != "project-id" {
					t.Errorf("request %d: X-Appwrite-Project = %q, want project-id", n, got.project)
				}
				checkBodyExactly(t, n, got.body, want.body)
			}
		})
	}
}

// TestRuleRoutesOmitUnset checks what is left out: a create for an api rule
// has no resource ID to send, and a rule without conditions sends none,
// because the API rejects an empty list.
func TestRuleRoutesOmitUnset(t *testing.T) {
	t.Parallel()
	server, requests := ruleServer(t)
	client := testWafClient(server.URL)

	spec := ruleSpec{ID: "rule-id", Type: typeDeny, Name: "deny all", ResourceType: resourceTypeAPI, Enabled: true}
	if _, err := createRule(client, spec); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := updateRule(client, spec); err != nil {
		t.Fatalf("update: %v", err)
	}
	for n, got := range *requests {
		if _, sent := got.body["conditions"]; sent {
			t.Errorf("request %d sent conditions for a rule that has none: %v", n, got.body["conditions"])
		}
	}
	if _, sent := (*requests)[0].body["resourceId"]; sent {
		t.Errorf("create sent a resource ID for an api rule: %v", (*requests)[0].body)
	}
}

// TestRuleRoutesFunctionResource checks that a function-scoped rule names its
// function on both routes.
func TestRuleRoutesFunctionResource(t *testing.T) {
	t.Parallel()
	server, requests := ruleServer(t)
	client := testWafClient(server.URL)

	spec := ruleSpec{ID: "rule-id", Type: typeBypass, Name: "office", ResourceType: resourceTypeFunctions,
		ResourceID: "function-id", Enabled: true}
	if _, err := createRule(client, spec); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := updateRule(client, spec); err != nil {
		t.Fatalf("update: %v", err)
	}
	for n, got := range *requests {
		if got.body["resourceType"] != "functions" || got.body["resourceId"] != "function-id" {
			t.Errorf("request %d: resourceType %v, resourceId %v, want functions/function-id",
				n, got.body["resourceType"], got.body["resourceId"])
		}
	}
}

func merge(maps ...map[string]any) map[string]any {
	out := map[string]any{}
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// checkBodyExactly fails on a missing, wrong or unexpected argument. An
// argument a route does not take is as much a wiring bug as a missing one.
func checkBodyExactly(t *testing.T, n int, body, want map[string]any) {
	t.Helper()
	var problems []string
	for key, value := range want {
		got, ok := body[key]
		if !ok {
			problems = append(problems, "missing "+key)
			continue
		}
		gotJSON, _ := json.Marshal(got)
		wantJSON, _ := json.Marshal(value)
		if string(gotJSON) != string(wantJSON) {
			problems = append(problems, key+" = "+string(gotJSON)+", want "+string(wantJSON))
		}
	}
	for key := range body {
		if _, ok := want[key]; !ok {
			problems = append(problems, "unexpected "+key)
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		t.Errorf("request %d body %v:\n  %s", n, body, strings.Join(problems, "\n  "))
	}
}
