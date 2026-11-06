package waf

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/sdk-for-go/v7/waf"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

const resourceName = "appwrite_waf_rule"

// WAF rules belong to a project and are authorized with a project key's
// wafRules.* scopes.
var scopes = []string{"wafRules.read", "wafRules.write"}

func wafClient(clients *common.AppwriteClients, projectID string) (*waf.Waf, error) {
	if err := common.ValidateProjectCredential(clients, resourceName, scopes...); err != nil {
		return nil, err
	}
	return appwrite.NewWaf(clients.ClientForProject(projectID)), nil
}

// Rule types, spelled as the API's action field spells them. Each has its own
// create and update route.
const (
	typeBypass    = "bypass"
	typeChallenge = "challenge"
	typeDeny      = "deny"
	typeRateLimit = "rateLimit"
	typeRedirect  = "redirect"
)

var ruleTypes = []string{typeBypass, typeChallenge, typeDeny, typeRateLimit, typeRedirect}

// What a rule can be scoped to. An api rule covers the whole project API and
// takes no resource ID; the others name a function or site.
const (
	resourceTypeAPI       = "api"
	resourceTypeFunctions = "functions"
	resourceTypeSites     = "sites"
)

var resourceTypes = []string{resourceTypeAPI, resourceTypeFunctions, resourceTypeSites}

const (
	strategyFixedWindow   = "fixedWindow"
	strategySlidingWindow = "slidingWindow"
	strategyTokenBucket   = "tokenBucket"
)

var strategies = []string{strategyFixedWindow, strategySlidingWindow, strategyTokenBucket}

// Limits and defaults the server applies. Defaults matter because the
// type-specific arguments are planned with them when left unset, so the plan
// shows what the server will store.
const (
	maxIDLength          = 36
	maxNameLength        = 128
	maxDescriptionLength = 1024
	maxResourceIDLength  = 255
	minPriority          = -100000
	maxPriority          = 100000

	challengeTypeCompute = "compute"
	minDifficulty        = 1
	maxDifficulty        = 5
	defaultDifficulty    = 3
	minChallengeTTL      = 900
	maxChallengeTTL      = 86400
	defaultChallengeTTL  = 1800

	minLimit         = 1
	maxLimit         = 1000000
	minInterval      = 1
	maxInterval      = 86400
	maxMaxBucketSize = 1000000
	defaultKey       = "ip"

	maxLocationLength = 2048
	minStatusCode     = 300
	maxStatusCode     = 399

	// Appwrite counts nested conditions towards the limit on conditions,
	// and caps the length of each condition string.
	maxConditions      = 100
	maxConditionLength = 4096
)

// formatError explains a failed request. A 404 without a structured error
// type is what a server that has no WAF routes at all returns -- self-hosted
// Appwrite answers with an HTML page -- and printing that page helps nobody.
func formatError(err error) string {
	if isMissingRoute(err) {
		return "the WAF API is not served at this endpoint (status 404). WAF rules are available only on Appwrite Cloud."
	}
	return common.FormatErrorWithAuthGuidance(err, common.ProjectCredentialGuidance(resourceName, scopes...))
}

// isMissingRoute reports a 404 that is not an Appwrite API error: the route
// itself does not exist, as opposed to the rule or project it names.
func isMissingRoute(err error) bool {
	return common.IsNotFoundError(err) && common.ErrorType(err) == ""
}

// isRuleGone reports a 404 that the API itself returned, meaning the rule (or
// the project holding it) no longer exists. A missing route is not one: it
// says nothing about the rule, so it must not remove it from state.
func isRuleGone(err error) bool {
	return common.IsNotFoundError(err) && common.ErrorType(err) != ""
}

// apiRule is a WAF rule as every route returns it, whichever type it is.
//
// It is decoded from the raw response rather than read from the SDK's typed
// models so that conditions keep their exact JSON, and so that one mapping
// covers all five types.
type apiRule struct {
	ID           string            `json:"$id"`
	CreatedAt    string            `json:"$createdAt"`
	UpdatedAt    string            `json:"$updatedAt"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	ResourceType string            `json:"resourceType"`
	ResourceID   string            `json:"resourceId"`
	Action       string            `json:"action"`
	Priority     int64             `json:"priority"`
	Enabled      bool              `json:"enabled"`
	Conditions   []json.RawMessage `json:"conditions"`
	Config       json.RawMessage   `json:"config"`

	ChallengeType *string `json:"challengeType"`
	Difficulty    *int64  `json:"difficulty"`
	TTL           *int64  `json:"ttl"`
	Limit         *int64  `json:"limit"`
	Interval      *int64  `json:"interval"`
	Key           *string `json:"key"`
	Strategy      *string `json:"strategy"`
	MaxBucketSize *int64  `json:"maxBucketSize"`
	Location      *string `json:"location"`
	StatusCode    *int64  `json:"statusCode"`
}

// ruleConfig is the action-specific configuration the server stores. The
// top-level fields are derived from it, but not on every route: reading a
// rate limit rule returns the default strategy and a zero bucket size at the
// top level whatever is stored. The configuration is the source of truth.
type ruleConfig struct {
	Type          *string `json:"type"`
	Difficulty    *int64  `json:"difficulty"`
	TTL           *int64  `json:"ttl"`
	Limit         *int64  `json:"limit"`
	Interval      *int64  `json:"interval"`
	Key           *string `json:"key"`
	Strategy      *string `json:"strategy"`
	MaxBucketSize *int64  `json:"maxBucketSize"`
	Location      *string `json:"location"`
	StatusCode    *int64  `json:"statusCode"`
}

// decodeRule reads a rule out of any WAF route's response.
func decodeRule(model models.Model) (*apiRule, error) {
	if isNilModel(model) {
		return nil, errors.New("empty response")
	}
	var rule apiRule
	if err := model.Decode(&rule); err != nil {
		return nil, fmt.Errorf("decoding WAF rule: %w", err)
	}

	// A rule type with no settings of its own returns its configuration as
	// [] rather than {}. Only an object carries anything.
	trimmed := bytes.TrimSpace(rule.Config)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return &rule, nil
	}
	var config ruleConfig
	if err := json.Unmarshal(trimmed, &config); err != nil {
		return nil, fmt.Errorf("decoding WAF rule configuration: %w", err)
	}
	rule.ChallengeType = prefer(config.Type, rule.ChallengeType)
	rule.Difficulty = prefer(config.Difficulty, rule.Difficulty)
	rule.TTL = prefer(config.TTL, rule.TTL)
	rule.Limit = prefer(config.Limit, rule.Limit)
	rule.Interval = prefer(config.Interval, rule.Interval)
	rule.Key = prefer(config.Key, rule.Key)
	rule.Strategy = prefer(config.Strategy, rule.Strategy)
	// The bucket size exists only for token bucket rules, and only in the
	// configuration. The 0 the top level reports otherwise is not a value.
	rule.MaxBucketSize = config.MaxBucketSize
	rule.Location = prefer(config.Location, rule.Location)
	rule.StatusCode = prefer(config.StatusCode, rule.StatusCode)
	return &rule, nil
}

// isNilModel catches a typed nil pointer inside the interface, which a plain
// nil comparison does not.
func isNilModel(model models.Model) bool {
	if model == nil {
		return true
	}
	v := reflect.ValueOf(model)
	return v.Kind() == reflect.Pointer && v.IsNil()
}

func prefer[T any](first, second *T) *T {
	if first != nil {
		return first
	}
	return second
}

// Logical condition methods nest other conditions in their values.
var logicalMethods = map[string]bool{"and": true, "or": true}

// normalizeCondition returns a condition in the form Appwrite stores it,
// encoded the way Terraform's jsonencode encodes it.
//
// Appwrite parses each condition and stores only its method, its attribute
// when not empty, and its values, recursing into and/or. Anything else is
// dropped. The provider compares in that form so that the configuration's
// spelling -- spacing, key order, escaping -- is kept rather than shown as a
// change on every plan.
//
// The encoding matters when there is no configuration to keep, as after an
// import. jsonencode sorts keys and escapes <, > and & as Go's encoder does,
// so writing the same bytes means a condition written with jsonencode imports
// without a diff.
func normalizeCondition(raw []byte) (string, error) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return "", err
	}
	normalized, err := normalizeConditionValue(value)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func normalizeConditionValue(value any) (map[string]any, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("a condition must be a JSON object")
	}
	method, _ := object["method"].(string)
	result := map[string]any{"method": method}
	if attribute, ok := object["attribute"].(string); ok && attribute != "" {
		result["attribute"] = attribute
	}

	values, _ := object["values"].([]any)
	if values == nil {
		values = []any{}
	}
	if logicalMethods[method] {
		nested := make([]any, 0, len(values))
		for _, v := range values {
			n, err := normalizeConditionValue(v)
			if err != nil {
				return nil, err
			}
			nested = append(nested, n)
		}
		values = nested
	}
	result["values"] = values
	return result, nil
}

// validateCondition checks a condition's structure: a JSON object with a
// method, an optional string attribute, and an array of values, where a
// logical condition (and, or) nests at least one further condition. It
// returns how many conditions the string holds, counting nested ones, which
// is what Appwrite's limit counts.
//
// Which methods and attributes exist is left to the server. Both lists grow
// on Appwrite's schedule, and a copy here would reject a new one until the
// next provider release.
func validateCondition(raw string) (int, error) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader([]byte(raw)))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return 0, fmt.Errorf("not valid JSON: %w", err)
	}
	if decoder.More() {
		return 0, errors.New("unexpected data after the condition object")
	}
	return validateConditionValue(value)
}

func validateConditionValue(value any) (int, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return 0, errors.New("a condition must be a JSON object such as {\"method\":\"equal\",\"attribute\":\"ip\",\"values\":[\"198.51.100.10\"]}")
	}
	method, ok := object["method"].(string)
	if !ok || method == "" {
		return 0, errors.New("a condition needs a method, such as equal or startsWith")
	}
	if attribute, present := object["attribute"]; present {
		if _, ok := attribute.(string); !ok {
			return 0, errors.New("a condition's attribute must be a string")
		}
	}
	values, present := object["values"]
	if present {
		if _, ok := values.([]any); !ok {
			return 0, errors.New("a condition's values must be an array")
		}
	}
	if !logicalMethods[method] {
		return 1, nil
	}

	nested, _ := values.([]any)
	if len(nested) == 0 {
		return 0, fmt.Errorf("an %s condition needs at least one condition in its values", method)
	}
	count := 1
	for _, n := range nested {
		c, err := validateConditionValue(n)
		if err != nil {
			return 0, err
		}
		count += c
	}
	return count, nil
}
