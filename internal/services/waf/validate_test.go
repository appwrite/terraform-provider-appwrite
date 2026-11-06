package waf

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// nullModel is a configuration that sets only the type and name.
func nullModel(ruleType string) ruleResourceModel {
	return ruleResourceModel{
		ID:            types.StringNull(),
		Type:          types.StringValue(ruleType),
		Name:          types.StringValue("rule"),
		Description:   types.StringNull(),
		ResourceType:  types.StringNull(),
		ResourceID:    types.StringNull(),
		Priority:      types.Int64Null(),
		Enabled:       types.BoolNull(),
		Conditions:    types.ListNull(types.StringType),
		ChallengeType: types.StringNull(),
		Difficulty:    types.Int64Null(),
		TTL:           types.Int64Null(),
		Limit:         types.Int64Null(),
		Interval:      types.Int64Null(),
		Key:           types.StringNull(),
		Strategy:      types.StringNull(),
		MaxBucketSize: types.Int64Null(),
		Location:      types.StringNull(),
		StatusCode:    types.Int64Null(),
		ProjectID:     types.StringNull(),
	}
}

func TestValidateRuleConfig(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	i := types.Int64Value
	s := types.StringValue

	conditions := func(values ...string) types.List {
		list, _ := types.ListValueFrom(ctx, types.StringType, values)
		return list
	}

	for name, tc := range map[string]struct {
		ruleType string
		edit     func(*ruleResourceModel)
		// wantErrors lists a fragment of each expected error; none means
		// the configuration is valid.
		wantErrors []string
	}{
		"bypass alone":            {ruleType: typeBypass},
		"deny alone":              {ruleType: typeDeny},
		"challenge with defaults": {ruleType: typeChallenge},
		"challenge with settings": {ruleType: typeChallenge, edit: func(m *ruleResourceModel) {
			m.Difficulty, m.TTL, m.ChallengeType = i(5), i(900), s("compute")
		}},
		"rate limit": {ruleType: typeRateLimit, edit: func(m *ruleResourceModel) { m.Limit, m.Interval = i(10), i(60) }},
		"rate limit without limit and interval": {ruleType: typeRateLimit,
			wantErrors: []string{"rateLimit rules require limit", "rateLimit rules require interval"}},
		"token bucket with bucket size": {ruleType: typeRateLimit, edit: func(m *ruleResourceModel) {
			m.Limit, m.Interval, m.Strategy, m.MaxBucketSize = i(10), i(60), s(strategyTokenBucket), i(50)
		}},
		"bucket size without token bucket": {ruleType: typeRateLimit, edit: func(m *ruleResourceModel) {
			m.Limit, m.Interval, m.MaxBucketSize = i(10), i(60), i(50)
		}, wantErrors: []string{"applies only when strategy is tokenBucket"}},
		"bucket size with an unknown strategy": {ruleType: typeRateLimit, edit: func(m *ruleResourceModel) {
			m.Limit, m.Interval, m.Strategy, m.MaxBucketSize = i(10), i(60), types.StringUnknown(), i(50)
		}},
		"redirect": {ruleType: typeRedirect, edit: func(m *ruleResourceModel) { m.Location, m.StatusCode = s("/"), i(302) }},
		"redirect without location": {ruleType: typeRedirect, edit: func(m *ruleResourceModel) { m.StatusCode = i(302) },
			wantErrors: []string{"redirect rules require location"}},
		"redirect location unknown until apply": {ruleType: typeRedirect, edit: func(m *ruleResourceModel) {
			m.Location, m.StatusCode = types.StringUnknown(), i(302)
		}},
		"rate limit settings on a deny rule": {ruleType: typeDeny, edit: func(m *ruleResourceModel) {
			m.Limit, m.Key = i(10), s("ip")
		}, wantErrors: []string{"limit is used only by rateLimit rules", "key is used only by rateLimit rules"}},
		"challenge settings on a redirect rule": {ruleType: typeRedirect, edit: func(m *ruleResourceModel) {
			m.Location, m.StatusCode, m.Difficulty = s("/"), i(302), i(2)
		}, wantErrors: []string{"difficulty is used only by challenge rules"}},
		"redirect settings on a bypass rule": {ruleType: typeBypass, edit: func(m *ruleResourceModel) {
			m.StatusCode = i(301)
		}, wantErrors: []string{"status_code is used only by redirect rules"}},

		"function rule": {ruleType: typeDeny, edit: func(m *ruleResourceModel) {
			m.ResourceType, m.ResourceID = s(resourceTypeFunctions), s("fn")
		}},
		"site rule without a site": {ruleType: typeDeny, edit: func(m *ruleResourceModel) {
			m.ResourceType = s(resourceTypeSites)
		}, wantErrors: []string{"resource_id is required when resource_type is sites"}},
		"api rule with a resource ID": {ruleType: typeDeny, edit: func(m *ruleResourceModel) {
			m.ResourceID = s("fn")
		}, wantErrors: []string{"resource_id is not used when resource_type is api"}},

		"valid conditions": {ruleType: typeDeny, edit: func(m *ruleResourceModel) {
			m.Conditions = conditions(`{"method":"equal","attribute":"ip","values":["10.0.0.1"]}`)
		}},
		"invalid condition": {ruleType: typeDeny, edit: func(m *ruleResourceModel) {
			m.Conditions = conditions(`{"method":"equal","attribute":"ip","values":["10.0.0.1"]}`, `ip == 10.0.0.1`)
		}, wantErrors: []string{"not valid JSON"}},
		"too many nested conditions": {ruleType: typeDeny, edit: func(m *ruleResourceModel) {
			nested := make([]string, 0, maxConditions)
			for range maxConditions {
				nested = append(nested, `{"method":"equal","attribute":"ip","values":["10.0.0.1"]}`)
			}
			m.Conditions = conditions(`{"method":"or","values":[` + strings.Join(nested, ",") + `]}`)
		}, wantErrors: []string{"at most 100 conditions"}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			model := nullModel(tc.ruleType)
			if tc.edit != nil {
				tc.edit(&model)
			}
			diags := validateRuleConfig(ctx, model)

			var got []string
			for _, d := range diags.Errors() {
				got = append(got, d.Detail())
			}
			if len(got) != len(tc.wantErrors) {
				t.Fatalf("got errors %q, want %d matching %q", got, len(tc.wantErrors), tc.wantErrors)
			}
			for _, want := range tc.wantErrors {
				found := false
				for _, g := range got {
					found = found || strings.Contains(g, want)
				}
				if !found {
					t.Errorf("no error containing %q in %q", want, got)
				}
			}
		})
	}
}
