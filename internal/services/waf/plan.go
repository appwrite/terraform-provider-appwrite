package waf

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Type-specific arguments default only for the rule type that uses them and
// are null for every other type, which a schema Default cannot express. These
// modifiers fill in the planned value when the configuration leaves it out,
// so the plan shows what the server will store and removing an argument
// plans it back to its default.

// configuredType returns the rule type from configuration, and false when it
// is not known yet.
func configuredType(ctx context.Context, config tfsdk.Config) (string, bool) {
	var ruleType types.String
	config.GetAttribute(ctx, path.Root("type"), &ruleType)
	if ruleType.IsUnknown() {
		return "", false
	}
	return ruleType.ValueString(), true
}

type typeDefaultString struct {
	ruleType string
	value    string
}

func defaultStringFor(ruleType, value string) planmodifier.String {
	return typeDefaultString{ruleType: ruleType, value: value}
}

func (m typeDefaultString) Description(context.Context) string {
	return fmt.Sprintf("Defaults to %q for %s rules and is null for other types.", m.value, m.ruleType)
}

func (m typeDefaultString) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m typeDefaultString) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}
	ruleType, known := configuredType(ctx, req.Config)
	switch {
	case !known:
		resp.PlanValue = types.StringUnknown()
	case ruleType == m.ruleType:
		resp.PlanValue = types.StringValue(m.value)
	default:
		resp.PlanValue = types.StringNull()
	}
}

type typeDefaultInt64 struct {
	ruleType string
	value    int64
}

func defaultInt64For(ruleType string, value int64) planmodifier.Int64 {
	return typeDefaultInt64{ruleType: ruleType, value: value}
}

func (m typeDefaultInt64) Description(context.Context) string {
	return fmt.Sprintf("Defaults to %d for %s rules and is null for other types.", m.value, m.ruleType)
}

func (m typeDefaultInt64) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m typeDefaultInt64) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if !req.ConfigValue.IsNull() {
		return
	}
	ruleType, known := configuredType(ctx, req.Config)
	switch {
	case !known:
		resp.PlanValue = types.Int64Unknown()
	case ruleType == m.ruleType:
		resp.PlanValue = types.Int64Value(m.value)
	default:
		resp.PlanValue = types.Int64Null()
	}
}

// maxBucketSizeDefault plans max_bucket_size as the server sets it when it is
// not given: equal to limit for a token bucket rule, absent otherwise.
type maxBucketSizeDefault struct{}

func (maxBucketSizeDefault) Description(context.Context) string {
	return "Defaults to limit for token bucket rate limit rules and is null otherwise."
}

func (m maxBucketSizeDefault) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (maxBucketSizeDefault) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if !req.ConfigValue.IsNull() {
		return
	}
	ruleType, known := configuredType(ctx, req.Config)
	if !known {
		resp.PlanValue = types.Int64Unknown()
		return
	}
	if ruleType != typeRateLimit {
		resp.PlanValue = types.Int64Null()
		return
	}

	var strategy types.String
	var limit types.Int64
	req.Config.GetAttribute(ctx, path.Root("strategy"), &strategy)
	req.Config.GetAttribute(ctx, path.Root("limit"), &limit)
	switch {
	case strategy.IsUnknown():
		resp.PlanValue = types.Int64Unknown()
	case strategy.ValueString() != strategyTokenBucket:
		// A null strategy is the default fixed window.
		resp.PlanValue = types.Int64Null()
	case limit.IsUnknown():
		resp.PlanValue = types.Int64Unknown()
	default:
		resp.PlanValue = limit
	}
}

// replaceWhenConditionsCleared replaces a rule whose conditions are removed
// from the configuration. The API rejects an empty conditions list, so an
// update can change a rule's conditions but never take all of them away.
func replaceWhenConditionsCleared() planmodifier.List {
	return listplanmodifier.RequiresReplaceIf(
		func(_ context.Context, req planmodifier.ListRequest, resp *listplanmodifier.RequiresReplaceIfFuncResponse) {
			hadConditions := !req.StateValue.IsNull() && !req.StateValue.IsUnknown() && len(req.StateValue.Elements()) > 0
			hasConditions := !req.PlanValue.IsNull() && (req.PlanValue.IsUnknown() || len(req.PlanValue.Elements()) > 0)
			resp.RequiresReplace = hadConditions && !hasConditions
		},
		"Removing every condition from a rule that has some replaces the rule.",
		"Removing every condition from a rule that has some replaces the rule.",
	)
}

var (
	_ planmodifier.String = typeDefaultString{}
	_ planmodifier.Int64  = typeDefaultInt64{}
	_ planmodifier.Int64  = maxBucketSizeDefault{}
)
