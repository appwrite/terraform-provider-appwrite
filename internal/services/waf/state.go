package waf

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// specFromModel reduces a planned rule to the values the API takes.
func specFromModel(ctx context.Context, model ruleResourceModel) (ruleSpec, diag.Diagnostics) {
	var diags diag.Diagnostics
	spec := ruleSpec{
		ID:           model.ID.ValueString(),
		Type:         model.Type.ValueString(),
		Name:         model.Name.ValueString(),
		Description:  model.Description.ValueString(),
		ResourceType: model.ResourceType.ValueString(),
		ResourceID:   model.ResourceID.ValueString(),
		Priority:     int(model.Priority.ValueInt64()),
		Enabled:      model.Enabled.ValueBool(),

		ChallengeType: model.ChallengeType.ValueString(),
		Difficulty:    int(model.Difficulty.ValueInt64()),
		TTL:           int(model.TTL.ValueInt64()),

		Limit:    int(model.Limit.ValueInt64()),
		Interval: int(model.Interval.ValueInt64()),
		Key:      model.Key.ValueString(),
		Strategy: model.Strategy.ValueString(),

		Location:   model.Location.ValueString(),
		StatusCode: int(model.StatusCode.ValueInt64()),
	}
	if spec.ResourceType == "" {
		spec.ResourceType = resourceTypeAPI
	}
	if !model.MaxBucketSize.IsNull() && !model.MaxBucketSize.IsUnknown() {
		size := int(model.MaxBucketSize.ValueInt64())
		spec.MaxBucketSize = &size
	}
	if !model.Conditions.IsNull() && !model.Conditions.IsUnknown() {
		var conditions []string
		diags.Append(model.Conditions.ElementsAs(ctx, &conditions, false)...)
		if len(conditions) > 0 {
			spec.Conditions = conditions
		}
	}
	return spec, diags
}

// mapRuleToState copies a rule into the model. Values the configuration
// controls are kept as configured when the server holds the same thing, so a
// condition written with different spacing or key order is not a diff.
func mapRuleToState(ctx context.Context, rule *apiRule, model *ruleResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(rule.ID)
	model.Type = types.StringValue(rule.Action)
	model.Name = types.StringValue(rule.Name)
	model.ResourceType = types.StringValue(rule.ResourceType)
	model.Priority = types.Int64Value(rule.Priority)
	model.Enabled = types.BoolValue(rule.Enabled)
	model.CreatedAt = types.StringValue(rule.CreatedAt)
	model.UpdatedAt = types.StringValue(rule.UpdatedAt)

	// The API returns "" for an unset description and for the resource ID
	// of an api rule. An unset argument stays null rather than becoming ""
	// and showing as a diff.
	model.Description = optionalString(model.Description, rule.Description)
	model.ResourceID = optionalString(model.ResourceID, rule.ResourceID)

	conditions, d := mergeConditions(ctx, model.Conditions, rule.Conditions)
	diags.Append(d...)
	model.Conditions = conditions

	model.ChallengeType, model.Difficulty, model.TTL = types.StringNull(), types.Int64Null(), types.Int64Null()
	model.Limit, model.Interval, model.MaxBucketSize = types.Int64Null(), types.Int64Null(), types.Int64Null()
	model.Key, model.Strategy = types.StringNull(), types.StringNull()
	model.Location, model.StatusCode = types.StringNull(), types.Int64Null()

	switch rule.Action {
	case typeChallenge:
		model.ChallengeType = types.StringValue(valueOr(rule.ChallengeType, challengeTypeCompute))
		model.Difficulty = types.Int64Value(valueOr(rule.Difficulty, defaultDifficulty))
		model.TTL = types.Int64Value(valueOr(rule.TTL, defaultChallengeTTL))
	case typeRateLimit:
		model.Limit = types.Int64PointerValue(rule.Limit)
		model.Interval = types.Int64PointerValue(rule.Interval)
		model.Key = types.StringValue(valueOr(rule.Key, defaultKey))
		model.Strategy = types.StringValue(valueOr(rule.Strategy, strategyFixedWindow))
		if model.Strategy.ValueString() == strategyTokenBucket {
			model.MaxBucketSize = types.Int64PointerValue(rule.MaxBucketSize)
		}
	case typeRedirect:
		model.Location = types.StringPointerValue(rule.Location)
		model.StatusCode = types.Int64PointerValue(rule.StatusCode)
	}
	return diags
}

func optionalString(current types.String, value string) types.String {
	if value == "" && current.IsNull() {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func valueOr[T any](value *T, fallback T) T {
	if value == nil {
		return fallback
	}
	return *value
}

// mergeConditions builds the conditions list from what the server returned,
// keeping each configured string whose condition the server holds unchanged.
//
// A rule without conditions comes back as an empty list, which is stored as
// null: the argument cannot be set to an empty list, so null is the only way
// the configuration can say "none".
func mergeConditions(ctx context.Context, current types.List, returned []json.RawMessage) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(returned) == 0 {
		return types.ListNull(types.StringType), diags
	}

	var configured []string
	if !current.IsNull() && !current.IsUnknown() {
		diags.Append(current.ElementsAs(ctx, &configured, false)...)
	}

	result := make([]string, len(returned))
	for i, raw := range returned {
		stored, err := normalizeCondition(raw)
		if err != nil {
			result[i] = compactJSON(raw)
			continue
		}
		result[i] = stored
		if i >= len(configured) {
			continue
		}
		if wanted, err := normalizeCondition([]byte(configured[i])); err == nil && wanted == stored {
			result[i] = configured[i]
		}
	}

	list, d := types.ListValueFrom(ctx, types.StringType, result)
	diags.Append(d...)
	return list, diags
}

func compactJSON(raw []byte) string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		return string(raw)
	}
	return buf.String()
}
