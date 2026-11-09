package proxy

import (
	"context"
	"fmt"
	"strings"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/proxy"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &ruleResource{}
	_ resource.ResourceWithConfigure      = &ruleResource{}
	_ resource.ResourceWithImportState    = &ruleResource{}
	_ resource.ResourceWithValidateConfig = &ruleResource{}
)

type ruleResource struct {
	clients *common.AppwriteClients
}

type ruleResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Domain               types.String `tfsdk:"domain"`
	Type                 types.String `tfsdk:"type"`
	ResourceID           types.String `tfsdk:"resource_id"`
	Branch               types.String `tfsdk:"branch"`
	RedirectURL          types.String `tfsdk:"redirect_url"`
	RedirectStatusCode   types.Int64  `tfsdk:"redirect_status_code"`
	RedirectResourceType types.String `tfsdk:"redirect_resource_type"`
	Status               types.String `tfsdk:"status"`
	Logs                 types.String `tfsdk:"logs"`
	RenewAt              types.String `tfsdk:"renew_at"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
	ProjectID            types.String `tfsdk:"project_id"`
}

func NewRuleResource() resource.Resource {
	return &ruleResource{}
}

func (r *ruleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_proxy_rule"
}

func (r *ruleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Manages a custom domain proxy rule: serving an Appwrite site, function or the Appwrite API on a domain, or redirecting a domain to a URL.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The proxy rule ID.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"domain": schema.StringAttribute{
				Description:   "The custom domain name. " + common.ForcesReplacementNote,
				Required:      true,
				PlanModifiers: replace,
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"type": schema.StringAttribute{
				Description: "What the rule does: `site` or `function` serves that resource on the domain, `api` serves the Appwrite API, and `redirect` redirects to `redirect_url`. " +
					common.ForcesReplacementNote,
				Required:      true,
				PlanModifiers: replace,
				Validators:    []validator.String{stringvalidator.OneOf(ruleTypes...)},
			},
			"resource_id": schema.StringAttribute{
				Description: "The ID of the site or function served by this rule. For a `redirect` rule, the site or function the rule is listed under, whose deletion also deletes the rule. " +
					"Required for every type except `api`, which does not take it. " + common.ForcesReplacementNote,
				Optional:      true,
				PlanModifiers: replace,
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"branch": schema.StringAttribute{
				Description: "The VCS branch that updates the rule automatically. Only for `site` and `function` rules. " +
					common.ForcesReplacementNote,
				Optional:      true,
				PlanModifiers: replace,
			},
			"redirect_url": schema.StringAttribute{
				Description: "The absolute URL a `redirect` rule redirects to. Required for `redirect` rules and not used by any other type. " +
					common.ForcesReplacementNote,
				Optional:      true,
				PlanModifiers: replace,
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"redirect_status_code": schema.Int64Attribute{
				Description: "The HTTP status code a `redirect` rule responds with: 301, 302, 307 or 308. Required for `redirect` rules and not used by any other type. " +
					common.ForcesReplacementNote,
				Optional:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
				Validators:    []validator.Int64{int64validator.OneOf(redirectStatusCodes...)},
			},
			"redirect_resource_type": schema.StringAttribute{
				Description: "The type of the resource named by `resource_id` on a `redirect` rule: `site` or `function`. Required for `redirect` rules and not used by any other type. " +
					common.ForcesReplacementNote,
				Optional:      true,
				PlanModifiers: replace,
				Validators:    []validator.String{stringvalidator.OneOf(redirectResourceTypes...)},
			},
			"status": schema.StringAttribute{
				Description: "The domain verification status.",
				Computed:    true,
			},
			"logs": schema.StringAttribute{
				Description: "Domain verification or certificate generation logs.",
				Computed:    true,
			},
			"renew_at": schema.StringAttribute{
				Description: "The certificate auto-renewal timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The rule creation timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The rule last update timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"project_id": common.ProjectIDAttribute(),
		},
	}
}

// ValidateConfig checks at plan time that each rule type has the arguments it
// needs and none it does not, which the API would otherwise reject mid-apply.
func (r *ruleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config ruleResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.Type.IsUnknown() || config.Type.IsNull() {
		return
	}
	resp.Diagnostics.Append(validateRuleConfig(config)...)
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

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	proxyClient, err := r.client(projectID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	rule, err := createRule(proxyClient, plan)
	if err != nil {
		resp.Diagnostics.AddError("Error creating proxy rule", common.FormatErrorWithAuthGuidance(err, common.ProjectCredentialGuidance("appwrite_proxy_rule", "rules.read", "rules.write")))
		return
	}

	// The create response can contain placeholder values for certificate fields
	// (for example, renewAt may be "datetime"). Read the rule back so the initial
	// state matches subsequent refreshes and imports.
	rule, err = proxyClient.GetRule(rule.Id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading created proxy rule", common.FormatErrorWithAuthGuidance(err, common.ProjectCredentialGuidance("appwrite_proxy_rule", "rules.read", "rules.write")))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	mapRuleToState(rule, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ruleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ruleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	proxyClient, err := r.client(projectID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	rule, err := proxyClient.GetRule(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading proxy rule", common.FormatErrorWithAuthGuidance(err, common.ProjectCredentialGuidance("appwrite_proxy_rule", "rules.read", "rules.write")))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	mapRuleToState(rule, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ruleResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Proxy rules cannot be updated in place. Change a rule attribute to replace the rule.")
}

func (r *ruleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ruleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	proxyClient, err := r.client(projectID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	_, err = proxyClient.DeleteRule(state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting proxy rule", common.FormatErrorWithAuthGuidance(err, common.ProjectCredentialGuidance("appwrite_proxy_rule", "rules.read", "rules.write")))
	}
}

func (r *ruleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	switch len(parts) {
	case 1:
		if parts[0] == "" {
			resp.Diagnostics.AddError("Invalid import ID", "Expected rule_id or project_id/rule_id")
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[0])...)
	case 2:
		if parts[0] == "" || parts[1] == "" {
			resp.Diagnostics.AddError("Invalid import ID", "Expected rule_id or project_id/rule_id")
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	default:
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected rule_id or project_id/rule_id, got: %s", req.ID))
	}
}

func (r *ruleResource) client(projectID string) (*proxy.Proxy, error) {
	if err := common.ValidateProjectCredential(r.clients, "appwrite_proxy_rule", "rules.read", "rules.write"); err != nil {
		return nil, err
	}
	return appwrite.NewProxy(r.clients.ClientForProject(projectID)), nil
}
