package waf

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/sdk-for-go/v7/waf"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func testWafClient(url string) *waf.Waf {
	// No retries: the SDK client here is bare, so each scripted response is
	// seen by the code under test exactly once.
	return appwrite.NewWaf(appwrite.NewClient(appwrite.WithEndpoint(url+"/v1"), appwrite.WithProject("project-id")))
}

// rawModel stands in for an SDK response model: Decode is the only method
// decodeRule uses.
type rawModel string

func (m rawModel) Decode(value any) error { return json.Unmarshal([]byte(m), value) }

func TestDecodeRuleReadsTheConfiguration(t *testing.T) {
	t.Parallel()

	// What reading a token bucket rule returns: the top-level strategy and
	// bucket size are always fixedWindow and 0, and only the configuration holds
	// what is stored.
	rule, err := decodeRule(rawModel(`{
		"$id": "r", "action": "rateLimit", "limit": 100, "interval": 60, "key": "ip",
		"strategy": "fixedWindow", "maxBucketSize": 0,
		"config": {"limit": 100, "interval": 60, "key": "ip", "strategy": "tokenBucket", "maxBucketSize": 250}
	}`))
	if err != nil {
		t.Fatalf("decodeRule: %v", err)
	}
	if got := valueOr(rule.Strategy, ""); got != strategyTokenBucket {
		t.Errorf("strategy = %q, want tokenBucket from the configuration", got)
	}
	if rule.MaxBucketSize == nil || *rule.MaxBucketSize != 250 {
		t.Errorf("maxBucketSize = %v, want 250 from the configuration", rule.MaxBucketSize)
	}

	// A fixed window rule has no bucket size, whatever the top level says.
	rule, err = decodeRule(rawModel(`{"action": "rateLimit", "maxBucketSize": 0,
		"config": {"limit": 5, "interval": 1, "key": "ip", "strategy": "fixedWindow"}}`))
	if err != nil {
		t.Fatalf("decodeRule: %v", err)
	}
	if rule.MaxBucketSize != nil {
		t.Errorf("maxBucketSize = %d, want none for a fixed window rule", *rule.MaxBucketSize)
	}

	// The challenge configuration calls the challenge type "type".
	rule, err = decodeRule(rawModel(`{"action": "challenge", "config": {"type": "compute", "difficulty": 5, "ttl": 900}}`))
	if err != nil {
		t.Fatalf("decodeRule: %v", err)
	}
	if valueOr(rule.ChallengeType, "") != "compute" || valueOr(rule.Difficulty, 0) != 5 || valueOr(rule.TTL, 0) != 900 {
		t.Errorf("challenge settings = %v %v %v, want compute 5 900", rule.ChallengeType, rule.Difficulty, rule.TTL)
	}
}

func TestDecodeRuleAcceptsAnEmptyArrayConfiguration(t *testing.T) {
	t.Parallel()

	// A bypass or deny rule returns its configuration as [].
	rule, err := decodeRule(rawModel(`{"$id": "r", "action": "deny", "config": [], "location": "/fallback"}`))
	if err != nil {
		t.Fatalf("decodeRule: %v", err)
	}
	if rule.ID != "r" || valueOr(rule.Location, "") != "/fallback" {
		t.Errorf("decoded %+v", rule)
	}
}

func TestDecodeRuleRejectsANilModel(t *testing.T) {
	t.Parallel()
	var typed *models.WafRuleDeny
	if _, err := decodeRule(typed); err == nil {
		t.Error("decodeRule accepted a nil model")
	}
}

func TestNormalizeCondition(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		configured string
		stored     string
	}{
		"key order and spacing": {
			configured: `{ "values": ["198.51.100.10"], "attribute": "ip", "method": "equal" }`,
			stored:     `{"method":"equal","attribute":"ip","values":["198.51.100.10"]}`,
		},
		"escaped slashes": {
			configured: `{"method":"startsWith","attribute":"path","values":["/v1/graphql"]}`,
			stored:     `{"method":"startsWith","attribute":"path","values":["\/v1\/graphql"]}`,
		},
		"empty attribute and unknown keys are dropped": {
			configured: `{"method":"or","attribute":"","note":"x","values":[{"method":"equal","attribute":"country","values":["NL"]}]}`,
			stored:     `{"method":"or","values":[{"method":"equal","attribute":"country","values":["NL"]}]}`,
		},
		"missing values": {
			configured: `{"method":"isNull","attribute":"userAgent"}`,
			stored:     `{"method":"isNull","attribute":"userAgent","values":[]}`,
		},
		"numbers keep their digits": {
			configured: `{"method":"greaterThan","attribute":"sessionAge","values":[86400]}`,
			stored:     `{"method":"greaterThan","attribute":"sessionAge","values":[86400]}`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			a, err := normalizeCondition([]byte(tc.configured))
			if err != nil {
				t.Fatalf("configured: %v", err)
			}
			b, err := normalizeCondition([]byte(tc.stored))
			if err != nil {
				t.Fatalf("stored: %v", err)
			}
			if a != b {
				t.Errorf("configured normalizes to %s, stored to %s", a, b)
			}
		})
	}

	// The normalized form is what an import writes, so it is byte for byte
	// what jsonencode produces for the same condition -- as Terraform 1.16
	// printed jsonencode({ values = ["/a?b=1&c=<2>"], method = "equal",
	// attribute = "path" }) -- and an imported condition written with
	// jsonencode shows no diff.
	got, err := normalizeCondition([]byte(`{"method":"equal","attribute":"path","values":["\/a?b=1&c=<2>"]}`))
	if err != nil {
		t.Fatal(err)
	}
	escaped := func(hex string) string { return `\` + "u00" + hex }
	want := `{"attribute":"path","method":"equal","values":["/a?b=1` + escaped("26") + "c=" + escaped("3c") + "2" +
		escaped("3e") + `"]}`
	if got != want {
		t.Errorf("normalized = %s, want %s", got, want)
	}
}

func TestValidateCondition(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		condition string
		count     int
		err       string
	}{
		"simple":           {condition: `{"method":"equal","attribute":"ip","values":["10.0.0.0/8"]}`, count: 1},
		"no values":        {condition: `{"method":"isNotNull","attribute":"userAgent"}`, count: 1},
		"nested":           {condition: `{"method":"and","values":[{"method":"equal","attribute":"method","values":["POST"]},{"method":"or","values":[{"method":"equal","attribute":"country","values":["NL"]}]}]}`, count: 4},
		"not json":         {condition: `method=equal`, err: "not valid JSON"},
		"trailing data":    {condition: `{"method":"equal"} {}`, err: "unexpected data"},
		"array":            {condition: `["equal"]`, err: "JSON object"},
		"no method":        {condition: `{"attribute":"ip","values":["1.2.3.4"]}`, err: "needs a method"},
		"numeric method":   {condition: `{"method":1}`, err: "needs a method"},
		"bad attribute":    {condition: `{"method":"equal","attribute":["ip"]}`, err: "attribute must be a string"},
		"values not array": {condition: `{"method":"equal","attribute":"ip","values":"1.2.3.4"}`, err: "values must be an array"},
		"empty or":         {condition: `{"method":"or","values":[]}`, err: "at least one condition"},
		"or of a string":   {condition: `{"method":"or","values":["x"]}`, err: "JSON object"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			count, err := validateCondition(tc.condition)
			switch {
			case tc.err == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tc.err != "" && (err == nil || !strings.Contains(err.Error(), tc.err)):
				t.Fatalf("error = %v, want one containing %q", err, tc.err)
			case tc.err == "" && count != tc.count:
				t.Errorf("count = %d, want %d", count, tc.count)
			}
		})
	}
}

func TestMergeConditions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	configured, _ := types.ListValueFrom(ctx, types.StringType, []string{
		`{ "method": "equal", "attribute": "ip", "values": ["198.51.100.10"] }`,
		`{"method":"equal","attribute":"country","values":["NL"]}`,
	})
	returned := []json.RawMessage{
		json.RawMessage(`{"method":"equal","attribute":"ip","values":["198.51.100.10"]}`),
		// Changed outside Terraform.
		json.RawMessage(`{"method":"equal","attribute":"country","values":["BE"]}`),
		// Added outside Terraform.
		json.RawMessage(`{"method":"startsWith","attribute":"path","values":["\/admin"]}`),
	}

	merged, diags := mergeConditions(ctx, configured, returned)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	var got []string
	merged.ElementsAs(ctx, &got, false)
	want := []string{
		`{ "method": "equal", "attribute": "ip", "values": ["198.51.100.10"] }`,
		`{"attribute":"country","method":"equal","values":["BE"]}`,
		`{"attribute":"path","method":"startsWith","values":["/admin"]}`,
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("merged =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	// No conditions on the server is null, whatever the state held.
	merged, _ = mergeConditions(ctx, configured, nil)
	if !merged.IsNull() {
		t.Errorf("merged = %v, want null for a rule without conditions", merged)
	}
}

func TestMapRuleToStateNullsOtherTypesAndKeepsUnsetArgumentsNull(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	rule, err := decodeRule(rawModel(`{"$id": "r", "action": "challenge", "name": "n", "description": "",
		"resourceType": "api", "resourceId": "", "priority": 3, "enabled": true, "conditions": [],
		"config": {"type": "compute", "difficulty": 3, "ttl": 1800},
		"limit": 100, "location": "/"}`))
	if err != nil {
		t.Fatal(err)
	}
	model := ruleResourceModel{Description: types.StringNull(), ResourceID: types.StringNull(), Conditions: types.ListNull(types.StringType)}
	if diags := mapRuleToState(ctx, rule, &model); diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	if !model.Description.IsNull() || !model.ResourceID.IsNull() || !model.Conditions.IsNull() {
		t.Errorf("unset arguments became %v %v %v, want null", model.Description, model.ResourceID, model.Conditions)
	}
	if model.Difficulty.ValueInt64() != 3 || model.TTL.ValueInt64() != 1800 || model.ChallengeType.ValueString() != "compute" {
		t.Errorf("challenge settings = %v %v %v", model.ChallengeType, model.Difficulty, model.TTL)
	}
	if !model.Limit.IsNull() || !model.Location.IsNull() || !model.Key.IsNull() {
		t.Errorf("another type's settings leaked into state: limit %v, location %v, key %v", model.Limit, model.Location, model.Key)
	}

	// A description removed outside Terraform reads as "", which shows as a
	// diff against the configured value rather than vanishing.
	model.Description = types.StringValue("was set")
	_ = mapRuleToState(ctx, rule, &model)
	if model.Description.IsNull() || model.Description.ValueString() != "" {
		t.Errorf("description = %v, want \"\"", model.Description)
	}
}

// errorServer returns a server answering every request with the given
// status, content type and body.
func errorServer(t *testing.T, status int, contentType, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

// TestNotFoundClassification tells a rule that is gone apart from a server
// that has no WAF routes. Only the first may remove a rule from state: a
// self-hosted endpoint, or a Cloud endpoint mistyped, must not make Terraform
// forget rules that still exist.
func TestNotFoundClassification(t *testing.T) {
	t.Parallel()

	missingRule := errorServer(t, http.StatusNotFound, "application/json",
		`{"message":"Requested WAF rule could not be found.","code":404,"type":"waf_rule_not_found","version":"1"}`)
	_, err := testWafClient(missingRule.URL).GetRule("rule-id")
	if !isRuleGone(err) || isMissingRoute(err) {
		t.Errorf("a waf_rule_not_found error: gone %v, missing route %v; want gone", isRuleGone(err), isMissingRoute(err))
	}

	selfHosted := errorServer(t, http.StatusNotFound, "text/html; charset=UTF-8", `<html><body>404</body></html>`)
	_, err = testWafClient(selfHosted.URL).GetRule("rule-id")
	if isRuleGone(err) || !isMissingRoute(err) {
		t.Errorf("an HTML 404: gone %v, missing route %v; want missing route", isRuleGone(err), isMissingRoute(err))
	}
	if msg := formatError(err); !strings.Contains(msg, "only on Appwrite Cloud") || strings.Contains(msg, "<html>") {
		t.Errorf("formatError = %q, want the Cloud-only explanation without the HTML page", msg)
	}

	unauthorized := errorServer(t, http.StatusUnauthorized, "application/json",
		`{"message":"missing scope","code":401,"type":"general_unauthorized_scope","version":"1"}`)
	_, err = testWafClient(unauthorized.URL).GetRule("rule-id")
	if msg := formatError(err); !strings.Contains(msg, "wafRules.read and wafRules.write") {
		t.Errorf("formatError = %q, want the scopes the resource needs", msg)
	}
}
