package waf

import (
	"fmt"

	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/sdk-for-go/v7/waf"
)

// ruleSpec is a planned rule reduced to the values the API takes. The plan
// has already filled in every type-specific default, so a value that applies
// to the rule's type is always set here.
type ruleSpec struct {
	ID           string
	Type         string
	Name         string
	Description  string
	ResourceType string
	ResourceID   string
	Priority     int
	Enabled      bool
	// Conditions is nil when the rule has none. The API rejects an empty
	// list, so nil is never sent.
	Conditions []string

	ChallengeType string
	Difficulty    int
	TTL           int

	Limit    int
	Interval int
	Key      string
	Strategy string
	// MaxBucketSize is set only for token bucket rules; the API ignores it
	// for the other strategies.
	MaxBucketSize *int

	Location   string
	StatusCode int
}

// createRule sends the create for the rule's type. Each type has its own
// route, and rate limit and redirect rules take their required settings as
// positional arguments.
//
// The description is always sent: the API accepts it empty, and sending it
// keeps create and update the same request.
func createRule(client *waf.Waf, spec ruleSpec) (models.Model, error) {
	switch spec.Type {
	case typeBypass:
		opts := []waf.CreateBypassRuleOption{
			client.WithCreateBypassRuleDescription(spec.Description),
			client.WithCreateBypassRulePriority(spec.Priority),
			client.WithCreateBypassRuleEnabled(spec.Enabled),
		}
		if spec.ResourceID != "" {
			opts = append(opts, client.WithCreateBypassRuleResourceId(spec.ResourceID))
		}
		if spec.Conditions != nil {
			opts = append(opts, client.WithCreateBypassRuleConditions(spec.Conditions))
		}
		return client.CreateBypassRule(spec.ID, spec.ResourceType, spec.Name, opts...)
	case typeDeny:
		opts := []waf.CreateDenyRuleOption{
			client.WithCreateDenyRuleDescription(spec.Description),
			client.WithCreateDenyRulePriority(spec.Priority),
			client.WithCreateDenyRuleEnabled(spec.Enabled),
		}
		if spec.ResourceID != "" {
			opts = append(opts, client.WithCreateDenyRuleResourceId(spec.ResourceID))
		}
		if spec.Conditions != nil {
			opts = append(opts, client.WithCreateDenyRuleConditions(spec.Conditions))
		}
		return client.CreateDenyRule(spec.ID, spec.ResourceType, spec.Name, opts...)
	case typeChallenge:
		opts := []waf.CreateChallengeRuleOption{
			client.WithCreateChallengeRuleDescription(spec.Description),
			client.WithCreateChallengeRulePriority(spec.Priority),
			client.WithCreateChallengeRuleEnabled(spec.Enabled),
			client.WithCreateChallengeRuleChallengeType(spec.ChallengeType),
			client.WithCreateChallengeRuleDifficulty(spec.Difficulty),
			client.WithCreateChallengeRuleTtl(spec.TTL),
		}
		if spec.ResourceID != "" {
			opts = append(opts, client.WithCreateChallengeRuleResourceId(spec.ResourceID))
		}
		if spec.Conditions != nil {
			opts = append(opts, client.WithCreateChallengeRuleConditions(spec.Conditions))
		}
		return client.CreateChallengeRule(spec.ID, spec.ResourceType, spec.Name, opts...)
	case typeRateLimit:
		opts := []waf.CreateRateLimitRuleOption{
			client.WithCreateRateLimitRuleDescription(spec.Description),
			client.WithCreateRateLimitRulePriority(spec.Priority),
			client.WithCreateRateLimitRuleEnabled(spec.Enabled),
			client.WithCreateRateLimitRuleKey(spec.Key),
			client.WithCreateRateLimitRuleStrategy(spec.Strategy),
		}
		if spec.MaxBucketSize != nil {
			opts = append(opts, client.WithCreateRateLimitRuleMaxBucketSize(*spec.MaxBucketSize))
		}
		if spec.ResourceID != "" {
			opts = append(opts, client.WithCreateRateLimitRuleResourceId(spec.ResourceID))
		}
		if spec.Conditions != nil {
			opts = append(opts, client.WithCreateRateLimitRuleConditions(spec.Conditions))
		}
		return client.CreateRateLimitRule(spec.ID, spec.ResourceType, spec.Name, spec.Limit, spec.Interval, opts...)
	case typeRedirect:
		opts := []waf.CreateRedirectRuleOption{
			client.WithCreateRedirectRuleDescription(spec.Description),
			client.WithCreateRedirectRulePriority(spec.Priority),
			client.WithCreateRedirectRuleEnabled(spec.Enabled),
		}
		if spec.ResourceID != "" {
			opts = append(opts, client.WithCreateRedirectRuleResourceId(spec.ResourceID))
		}
		if spec.Conditions != nil {
			opts = append(opts, client.WithCreateRedirectRuleConditions(spec.Conditions))
		}
		return client.CreateRedirectRule(spec.ID, spec.ResourceType, spec.Name, spec.Location, spec.StatusCode, opts...)
	default:
		return nil, fmt.Errorf("unsupported WAF rule type %q", spec.Type)
	}
}

// updateRule sends the update for the rule's type.
//
// The update routes change only what they are sent, so everything is sent:
// a description removed from the configuration has to arrive as "" to be
// cleared, and a type-specific setting removed from it has to arrive as its
// default. The resource ID is sent with the resource type every time because
// the API clears the stored ID whenever a type arrives without one.
//
// Conditions are the exception. The API rejects an empty list, so a rule
// cannot lose all its conditions in place; the plan replaces it instead.
// The strategy of a rate limit rule cannot be updated either, for the same
// reason.
func updateRule(client *waf.Waf, spec ruleSpec) (models.Model, error) {
	switch spec.Type {
	case typeBypass:
		opts := []waf.UpdateBypassRuleOption{
			client.WithUpdateBypassRuleResourceType(spec.ResourceType),
			client.WithUpdateBypassRuleResourceId(spec.ResourceID),
			client.WithUpdateBypassRuleName(spec.Name),
			client.WithUpdateBypassRuleDescription(spec.Description),
			client.WithUpdateBypassRulePriority(spec.Priority),
			client.WithUpdateBypassRuleEnabled(spec.Enabled),
		}
		if spec.Conditions != nil {
			opts = append(opts, client.WithUpdateBypassRuleConditions(spec.Conditions))
		}
		return client.UpdateBypassRule(spec.ID, opts...)
	case typeDeny:
		opts := []waf.UpdateDenyRuleOption{
			client.WithUpdateDenyRuleResourceType(spec.ResourceType),
			client.WithUpdateDenyRuleResourceId(spec.ResourceID),
			client.WithUpdateDenyRuleName(spec.Name),
			client.WithUpdateDenyRuleDescription(spec.Description),
			client.WithUpdateDenyRulePriority(spec.Priority),
			client.WithUpdateDenyRuleEnabled(spec.Enabled),
		}
		if spec.Conditions != nil {
			opts = append(opts, client.WithUpdateDenyRuleConditions(spec.Conditions))
		}
		return client.UpdateDenyRule(spec.ID, opts...)
	case typeChallenge:
		opts := []waf.UpdateChallengeRuleOption{
			client.WithUpdateChallengeRuleResourceType(spec.ResourceType),
			client.WithUpdateChallengeRuleResourceId(spec.ResourceID),
			client.WithUpdateChallengeRuleName(spec.Name),
			client.WithUpdateChallengeRuleDescription(spec.Description),
			client.WithUpdateChallengeRulePriority(spec.Priority),
			client.WithUpdateChallengeRuleEnabled(spec.Enabled),
			client.WithUpdateChallengeRuleChallengeType(spec.ChallengeType),
			client.WithUpdateChallengeRuleDifficulty(spec.Difficulty),
			client.WithUpdateChallengeRuleTtl(spec.TTL),
		}
		if spec.Conditions != nil {
			opts = append(opts, client.WithUpdateChallengeRuleConditions(spec.Conditions))
		}
		return client.UpdateChallengeRule(spec.ID, opts...)
	case typeRateLimit:
		opts := []waf.UpdateRateLimitRuleOption{
			client.WithUpdateRateLimitRuleResourceType(spec.ResourceType),
			client.WithUpdateRateLimitRuleResourceId(spec.ResourceID),
			client.WithUpdateRateLimitRuleName(spec.Name),
			client.WithUpdateRateLimitRuleDescription(spec.Description),
			client.WithUpdateRateLimitRulePriority(spec.Priority),
			client.WithUpdateRateLimitRuleEnabled(spec.Enabled),
			client.WithUpdateRateLimitRuleLimit(spec.Limit),
			client.WithUpdateRateLimitRuleInterval(spec.Interval),
			client.WithUpdateRateLimitRuleKey(spec.Key),
		}
		if spec.MaxBucketSize != nil {
			opts = append(opts, client.WithUpdateRateLimitRuleMaxBucketSize(*spec.MaxBucketSize))
		}
		if spec.Conditions != nil {
			opts = append(opts, client.WithUpdateRateLimitRuleConditions(spec.Conditions))
		}
		return client.UpdateRateLimitRule(spec.ID, opts...)
	case typeRedirect:
		opts := []waf.UpdateRedirectRuleOption{
			client.WithUpdateRedirectRuleResourceType(spec.ResourceType),
			client.WithUpdateRedirectRuleResourceId(spec.ResourceID),
			client.WithUpdateRedirectRuleName(spec.Name),
			client.WithUpdateRedirectRuleDescription(spec.Description),
			client.WithUpdateRedirectRulePriority(spec.Priority),
			client.WithUpdateRedirectRuleEnabled(spec.Enabled),
			client.WithUpdateRedirectRuleLocation(spec.Location),
			client.WithUpdateRedirectRuleStatusCode(spec.StatusCode),
		}
		if spec.Conditions != nil {
			opts = append(opts, client.WithUpdateRedirectRuleConditions(spec.Conditions))
		}
		return client.UpdateRedirectRule(spec.ID, opts...)
	default:
		return nil, fmt.Errorf("unsupported WAF rule type %q", spec.Type)
	}
}
