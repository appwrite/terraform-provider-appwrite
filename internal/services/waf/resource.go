package waf

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/appwrite/sdk-for-go/v7/id"
	"github.com/appwrite/sdk-for-go/v7/waf"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

var (
	_ resource.Resource                   = &ruleResource{}
	_ resource.ResourceWithConfigure      = &ruleResource{}
	_ resource.ResourceWithImportState    = &ruleResource{}
	_ resource.ResourceWithValidateConfig = &ruleResource{}
	_ resource.ResourceWithModifyPlan     = &ruleResource{}
)

type ruleResource struct {
	clients *common.AppwriteClients
}

type ruleResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Type         types.String `tfsdk:"type"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	ResourceType types.String `tfsdk:"resource_type"`
	ResourceID   types.String `tfsdk:"resource_id"`
	Priority     types.Int64  `tfsdk:"priority"`
	Enabled      types.Bool   `tfsdk:"enabled"`
	Conditions   types.List   `tfsdk:"conditions"`

	ChallengeType types.String `tfsdk:"challenge_type"`
	Difficulty    types.Int64  `tfsdk:"difficulty"`
	TTL           types.Int64  `tfsdk:"ttl"`

	Limit         types.Int64  `tfsdk:"limit"`
	Interval      types.Int64  `tfsdk:"interval"`
	Key           types.String `tfsdk:"key"`
	Strategy      types.String `tfsdk:"strategy"`
	MaxBucketSize types.Int64  `tfsdk:"max_bucket_size"`

	Location   types.String `tfsdk:"location"`
	StatusCode types.Int64  `tfsdk:"status_code"`

	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
	ProjectID types.String `tfsdk:"project_id"`
}

func NewRuleResource() resource.Resource {
	return &ruleResource{}
}

func (r *ruleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_waf_rule"
}

// ruleIDPattern is Appwrite's rule for custom IDs.
var ruleIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func (r *ruleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a web application firewall (WAF) rule on an Appwrite Cloud project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The rule ID. Generated when not set. Rule IDs are unique across Appwrite Cloud, not " +
					"only within the project. " + common.ForcesReplacementNote,
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, maxIDLength),
					stringvalidator.RegexMatches(ruleIDPattern,
						"must contain only letters, digits, periods, hyphens and underscores, and must not start with a special character"),
				},
			},
			"type": schema.StringAttribute{
				Description: "What the rule does with a request it matches: " + strings.Join(ruleTypes, ", ") + ". " +
					common.ForcesReplacementNote,
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.OneOf(ruleTypes...)},
			},
			"name": schema.StringAttribute{
				Description: fmt.Sprintf("The rule name, up to %d characters.", maxNameLength),
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, maxNameLength)},
			},
			"description": schema.StringAttribute{
				Description: fmt.Sprintf("A description of the rule, up to %d characters.", maxDescriptionLength),
				Optional:    true,
				Validators:  []validator.String{stringvalidator.LengthAtMost(maxDescriptionLength)},
			},
			"resource_type": schema.StringAttribute{
				Description: "What the rule applies to: " + strings.Join(resourceTypes, ", ") + ". api covers " +
					"the whole project API. Defaults to api.",
				Optional:   true,
				Computed:   true,
				Default:    stringdefault.StaticString(resourceTypeAPI),
				Validators: []validator.String{stringvalidator.OneOf(resourceTypes...)},
			},
			"resource_id": schema.StringAttribute{
				Description: "The ID of the function or site the rule applies to. Required when resource_type " +
					"is functions or sites, and not accepted for api.",
				Optional:   true,
				Validators: []validator.String{stringvalidator.LengthBetween(1, maxResourceIDLength)},
			},
			"priority": schema.Int64Attribute{
				Description: fmt.Sprintf("The evaluation order, from %d to %d. Rules with lower values are "+
					"evaluated first, and the first rule that matches a request decides it. Defaults to 0.",
					minPriority, maxPriority),
				Optional:   true,
				Computed:   true,
				Default:    int64default.StaticInt64(0),
				Validators: []validator.Int64{int64validator.Between(minPriority, maxPriority)},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the rule is enforced. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"conditions": schema.ListAttribute{
				Description: fmt.Sprintf("The conditions a request must all meet for the rule to match, each a "+
					"JSON object usually written with jsonencode, such as "+
					"jsonencode({ method = \"equal\", attribute = \"ip\", values = [\"198.51.100.10\"] }). "+
					"An and or an or condition nests further conditions in values. A rule with no conditions matches "+
					"every request. Up to %d conditions, counting nested ones. Removing every condition from a "+
					"rule replaces it, because Appwrite cannot clear a rule's conditions in place.", maxConditions),
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					replaceWhenConditionsCleared(),
				},
				Validators: []validator.List{
					listvalidator.SizeBetween(1, maxConditions),
					listvalidator.ValueStringsAre(stringvalidator.LengthBetween(1, maxConditionLength)),
				},
			},

			"challenge_type": schema.StringAttribute{
				Description: "The challenge a matching visitor must pass. Only compute, a proof-of-work " +
					"challenge, is available. Defaults to compute for challenge rules; not accepted for other types.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{defaultStringFor(typeChallenge, challengeTypeCompute)},
				Validators:    []validator.String{stringvalidator.OneOf(challengeTypeCompute)},
			},
			"difficulty": schema.Int64Attribute{
				Description: fmt.Sprintf("The challenge difficulty, from %d (easiest) to %d (hardest). Defaults "+
					"to %d for challenge rules; not accepted for other types.", minDifficulty, maxDifficulty, defaultDifficulty),
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{defaultInt64For(typeChallenge, defaultDifficulty)},
				Validators:    []validator.Int64{int64validator.Between(minDifficulty, maxDifficulty)},
			},
			"ttl": schema.Int64Attribute{
				Description: fmt.Sprintf("How long, in seconds, a visitor who passed the challenge is let through "+
					"before being challenged again, from %d to %d. Defaults to %d for challenge rules; not "+
					"accepted for other types.", minChallengeTTL, maxChallengeTTL, defaultChallengeTTL),
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{defaultInt64For(typeChallenge, defaultChallengeTTL)},
				Validators:    []validator.Int64{int64validator.Between(minChallengeTTL, maxChallengeTTL)},
			},

			"limit": schema.Int64Attribute{
				Description: fmt.Sprintf("The number of matching requests allowed per interval, from %d to %d. "+
					"Required for rateLimit rules; not accepted for other types.", minLimit, maxLimit),
				Optional:   true,
				Validators: []validator.Int64{int64validator.Between(minLimit, maxLimit)},
			},
			"interval": schema.Int64Attribute{
				Description: fmt.Sprintf("The rate limit interval in seconds, from %d to %d. Required for "+
					"rateLimit rules; not accepted for other types.", minInterval, maxInterval),
				Optional:   true,
				Validators: []validator.Int64{int64validator.Between(minInterval, maxInterval)},
			},
			"key": schema.StringAttribute{
				Description: "What requests are counted by: ip counts per client IP, userId per signed-in user. " +
					"A userId rule does not apply to requests without a signed-in user. Defaults to ip for " +
					"rateLimit rules; not accepted for other types.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{defaultStringFor(typeRateLimit, defaultKey)},
				Validators:    []validator.String{stringvalidator.OneOf("ip", "userId")},
			},
			"strategy": schema.StringAttribute{
				Description: "How requests are counted: fixedWindow counts per interval, slidingWindow weights " +
					"in the previous interval for smoother limiting, and tokenBucket refills continuously at " +
					"limit per interval to allow short bursts. Defaults to fixedWindow for rateLimit rules; not " +
					"accepted for other types. " + common.ForcesReplacementNote,
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					defaultStringFor(typeRateLimit, strategyFixedWindow),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{stringvalidator.OneOf(strategies...)},
			},
			"max_bucket_size": schema.Int64Attribute{
				Description: fmt.Sprintf("The largest burst a tokenBucket rule allows, in requests, from %d to "+
					"%d. Defaults to limit. Accepted only when strategy is tokenBucket.", minLimit, maxMaxBucketSize),
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{maxBucketSizeDefault{}},
				Validators:    []validator.Int64{int64validator.Between(minLimit, maxMaxBucketSize)},
			},

			"location": schema.StringAttribute{
				Description: fmt.Sprintf("Where a redirect rule sends matching requests, as a path or URL of up "+
					"to %d characters. Required for redirect rules; not accepted for other types.", maxLocationLength),
				Optional:   true,
				Validators: []validator.String{stringvalidator.LengthBetween(1, maxLocationLength)},
			},
			"status_code": schema.Int64Attribute{
				Description: fmt.Sprintf("The HTTP status a redirect rule responds with, from %d to %d. Required "+
					"for redirect rules; not accepted for other types.", minStatusCode, maxStatusCode),
				Optional:   true,
				Validators: []validator.Int64{int64validator.Between(minStatusCode, maxStatusCode)},
			},

			"created_at": schema.StringAttribute{
				Description:   "The rule creation timestamp in ISO 8601 format.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Description:   "The rule last update timestamp in ISO 8601 format.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{common.UseStateForUnknownUnlessUpdating()},
			},
			"project_id": common.ProjectIDAttribute(),
		},
	}
}

// ValidateConfig catches at plan time what the API would otherwise reject or
// silently ignore mid-apply: a setting on a rule type that has no use for it,
// a missing one that the type requires, a resource ID that does not fit the
// resource type, and a condition that is not a condition.
func (r *ruleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config ruleResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(validateRuleConfig(ctx, config)...)
}

type typedField struct {
	name     string
	ruleType string
	null     bool
	unknown  bool
	required bool
}

func validateRuleConfig(ctx context.Context, config ruleResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	diags.Append(validateResource(config)...)
	diags.Append(validateConditions(ctx, config.Conditions)...)

	if config.Type.IsUnknown() || config.Type.IsNull() {
		return diags
	}
	ruleType := config.Type.ValueString()

	fields := []typedField{
		{"challenge_type", typeChallenge, config.ChallengeType.IsNull(), config.ChallengeType.IsUnknown(), false},
		{"difficulty", typeChallenge, config.Difficulty.IsNull(), config.Difficulty.IsUnknown(), false},
		{"ttl", typeChallenge, config.TTL.IsNull(), config.TTL.IsUnknown(), false},
		{"limit", typeRateLimit, config.Limit.IsNull(), config.Limit.IsUnknown(), true},
		{"interval", typeRateLimit, config.Interval.IsNull(), config.Interval.IsUnknown(), true},
		{"key", typeRateLimit, config.Key.IsNull(), config.Key.IsUnknown(), false},
		{"strategy", typeRateLimit, config.Strategy.IsNull(), config.Strategy.IsUnknown(), false},
		{"max_bucket_size", typeRateLimit, config.MaxBucketSize.IsNull(), config.MaxBucketSize.IsUnknown(), false},
		{"location", typeRedirect, config.Location.IsNull(), config.Location.IsUnknown(), true},
		{"status_code", typeRedirect, config.StatusCode.IsNull(), config.StatusCode.IsUnknown(), true},
	}
	for _, field := range fields {
		// A value from another resource may not be known until apply, and
		// may turn out null. Only a value known now can be judged.
		if field.unknown {
			continue
		}
		used := field.ruleType == ruleType
		switch {
		case used && field.required && field.null:
			diags.AddAttributeError(path.Root(field.name), "Missing required argument",
				fmt.Sprintf("%s rules require %s.", ruleType, field.name))
		case !used && !field.null:
			diags.AddAttributeError(path.Root(field.name), "Unsupported argument",
				fmt.Sprintf("%s is used only by %s rules, not %s rules. Remove it from the configuration.",
					field.name, field.ruleType, ruleType))
		}
	}

	// The API accepts a bucket size for any rate limit rule and ignores it
	// unless the strategy is tokenBucket. Ignoring a setting silently would
	// show as a perpetual diff, so it is refused instead.
	if ruleType == typeRateLimit && !config.MaxBucketSize.IsNull() && !config.MaxBucketSize.IsUnknown() &&
		!config.Strategy.IsUnknown() && config.Strategy.ValueString() != strategyTokenBucket {
		diags.AddAttributeError(path.Root("max_bucket_size"), "Unsupported argument",
			"max_bucket_size applies only when strategy is tokenBucket. Set strategy = \"tokenBucket\" or remove max_bucket_size.")
	}
	return diags
}

func validateResource(config ruleResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if config.ResourceType.IsUnknown() || config.ResourceID.IsUnknown() {
		return diags
	}
	resourceType := config.ResourceType.ValueString()
	if config.ResourceType.IsNull() {
		resourceType = resourceTypeAPI
	}
	switch {
	case resourceType == resourceTypeAPI && !config.ResourceID.IsNull():
		diags.AddAttributeError(path.Root("resource_id"), "Unsupported argument",
			"resource_id is not used when resource_type is api, which covers the whole project API. "+
				"Remove it, or set resource_type to functions or sites.")
	case resourceType != resourceTypeAPI && config.ResourceID.IsNull():
		diags.AddAttributeError(path.Root("resource_id"), "Missing required argument",
			fmt.Sprintf("resource_id is required when resource_type is %s.", resourceType))
	}
	return diags
}

func validateConditions(ctx context.Context, conditions types.List) diag.Diagnostics {
	var diags diag.Diagnostics
	if conditions.IsNull() || conditions.IsUnknown() {
		return diags
	}
	var elements []types.String
	diags.Append(conditions.ElementsAs(ctx, &elements, false)...)
	if diags.HasError() {
		return diags
	}

	total := 0
	for i, element := range elements {
		if element.IsNull() || element.IsUnknown() {
			continue
		}
		count, err := validateCondition(element.ValueString())
		if err != nil {
			diags.AddAttributeError(path.Root("conditions").AtListIndex(i), "Invalid condition", err.Error())
			continue
		}
		total += count
	}
	if total > maxConditions {
		diags.AddAttributeError(path.Root("conditions"), "Too many conditions",
			fmt.Sprintf("A rule holds at most %d conditions, counting those nested in logical conditions; this one has %d.",
				maxConditions, total))
	}
	return diags
}

// ModifyPlan marks updated_at unknown whenever the update will change
// anything.
//
// The shared updated_at modifier decides by comparing the attributes known in
// the proposed plan, and runs before the type-specific defaults are filled
// in. Removing difficulty from a challenge rule changes only a default, which
// is still unknown at that point, so the old timestamp would be planned and
// the server's new one rejected after apply. Here every attribute has its
// final planned value.
func (r *ruleResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var plan, state ruleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || plan.UpdatedAt.IsUnknown() {
		return
	}
	if planChangesRule(plan, state) {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("updated_at"), types.StringUnknown())...)
	}
}

func planChangesRule(plan, state ruleResourceModel) bool {
	pairs := []struct{ planned, prior attr.Value }{
		{plan.Name, state.Name}, {plan.Description, state.Description},
		{plan.ResourceType, state.ResourceType}, {plan.ResourceID, state.ResourceID},
		{plan.Priority, state.Priority}, {plan.Enabled, state.Enabled}, {plan.Conditions, state.Conditions},
		{plan.ChallengeType, state.ChallengeType}, {plan.Difficulty, state.Difficulty}, {plan.TTL, state.TTL},
		{plan.Limit, state.Limit}, {plan.Interval, state.Interval}, {plan.Key, state.Key},
		{plan.MaxBucketSize, state.MaxBucketSize},
		{plan.Location, state.Location}, {plan.StatusCode, state.StatusCode},
	}
	for _, pair := range pairs {
		if !pair.planned.Equal(pair.prior) {
			return true
		}
	}
	return false
}

func (r *ruleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*common.AppwriteClients)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *common.AppwriteClients, got: %T", req.ProviderData))
		return
	}
	r.clients = clients
}

func (r *ruleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ruleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, client, ok := r.client(plan.ProjectID, &resp.Diagnostics)
	if !ok {
		return
	}

	spec, diags := specFromModel(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		spec.ID = id.Unique()
	}

	response, err := createRule(client, spec)
	if err != nil {
		resp.Diagnostics.AddError("Error creating WAF rule", formatError(err))
		return
	}
	rule, err := decodeRule(response)
	if err != nil {
		resp.Diagnostics.AddError("Error reading created WAF rule", err.Error())
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	resp.Diagnostics.Append(mapRuleToState(ctx, rule, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ruleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ruleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, client, ok := r.client(state.ProjectID, &resp.Diagnostics)
	if !ok {
		return
	}

	response, err := client.GetRule(state.ID.ValueString())
	if err != nil {
		if isRuleGone(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading WAF rule", formatError(err))
		return
	}
	rule, err := decodeRule(response)
	if err != nil {
		resp.Diagnostics.AddError("Error reading WAF rule", err.Error())
		return
	}

	state.ProjectID = types.StringValue(projectID)
	resp.Diagnostics.Append(mapRuleToState(ctx, rule, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ruleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ruleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, client, ok := r.client(plan.ProjectID, &resp.Diagnostics)
	if !ok {
		return
	}

	plan.ID = state.ID
	spec, diags := specFromModel(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := updateRule(client, spec)
	if err != nil {
		resp.Diagnostics.AddError("Error updating WAF rule", formatError(err))
		return
	}
	rule, err := decodeRule(response)
	if err != nil {
		resp.Diagnostics.AddError("Error reading updated WAF rule", err.Error())
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	resp.Diagnostics.Append(mapRuleToState(ctx, rule, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ruleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ruleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, client, ok := r.client(state.ProjectID, &resp.Diagnostics)
	if !ok {
		return
	}

	_, err := client.DeleteRule(state.ID.ValueString())
	if err != nil && !isRuleGone(err) {
		resp.Diagnostics.AddError("Error deleting WAF rule", formatError(err))
	}
}

func (r *ruleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	const expected = "rule_id or project_id/rule_id"

	parts := strings.Split(req.ID, "/")
	for _, part := range parts {
		if part == "" {
			resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected %s, got: %s", expected, req.ID))
			return
		}
	}
	switch len(parts) {
	case 1:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[0])...)
	case 2:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	default:
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected %s, got: %s", expected, req.ID))
	}
}

func (r *ruleResource) client(projectID types.String, diags *diag.Diagnostics) (string, *waf.Waf, bool) {
	resolved, err := common.ResolveProjectID(r.clients, projectID)
	if err != nil {
		diags.AddError("Error resolving project ID", err.Error())
		return "", nil, false
	}
	client, err := wafClient(r.clients, resolved)
	if err != nil {
		diags.AddError("Unsupported Appwrite credential", err.Error())
		return "", nil, false
	}
	return resolved, client, true
}
