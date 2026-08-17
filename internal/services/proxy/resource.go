package proxy

import (
	"context"
	"fmt"
	"strings"

	"github.com/appwrite/sdk-for-go/v6/appwrite"
	"github.com/appwrite/sdk-for-go/v6/models"
	"github.com/appwrite/sdk-for-go/v6/proxy"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &ruleResource{}
	_ resource.ResourceWithConfigure   = &ruleResource{}
	_ resource.ResourceWithImportState = &ruleResource{}
)

type ruleResource struct {
	clients *common.AppwriteClients
}

type ruleResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Domain     types.String `tfsdk:"domain"`
	Type       types.String `tfsdk:"type"`
	ResourceID types.String `tfsdk:"resource_id"`
	Branch     types.String `tfsdk:"branch"`
	Status     types.String `tfsdk:"status"`
	Logs       types.String `tfsdk:"logs"`
	RenewAt    types.String `tfsdk:"renew_at"`
	CreatedAt  types.String `tfsdk:"created_at"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
	ProjectID  types.String `tfsdk:"project_id"`
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
		Description: "Manages a custom domain proxy rule for an Appwrite site or function.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The proxy rule ID.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"domain": schema.StringAttribute{
				Description:   "The custom domain name.",
				Required:      true,
				PlanModifiers: replace,
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"type": schema.StringAttribute{
				Description:   "The target resource type: site or function.",
				Required:      true,
				PlanModifiers: replace,
				Validators:    []validator.String{stringvalidator.OneOf("site", "function")},
			},
			"resource_id": schema.StringAttribute{
				Description:   "The ID of the site or function served by this rule.",
				Required:      true,
				PlanModifiers: replace,
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"branch": schema.StringAttribute{
				Description:   "The VCS branch that updates the rule automatically.",
				Optional:      true,
				PlanModifiers: replace,
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
	proxyClient := appwrite.NewProxy(r.clients.ClientForProject(projectID))

	var rule *models.ProxyRule
	switch plan.Type.ValueString() {
	case "site":
		var opts []proxy.CreateSiteRuleOption
		if !plan.Branch.IsNull() && !plan.Branch.IsUnknown() {
			opts = append(opts, proxyClient.WithCreateSiteRuleBranch(plan.Branch.ValueString()))
		}
		rule, err = proxyClient.CreateSiteRule(plan.Domain.ValueString(), plan.ResourceID.ValueString(), opts...)
	case "function":
		var opts []proxy.CreateFunctionRuleOption
		if !plan.Branch.IsNull() && !plan.Branch.IsUnknown() {
			opts = append(opts, proxyClient.WithCreateFunctionRuleBranch(plan.Branch.ValueString()))
		}
		rule, err = proxyClient.CreateFunctionRule(plan.Domain.ValueString(), plan.ResourceID.ValueString(), opts...)
	default:
		resp.Diagnostics.AddError("Invalid proxy rule type", fmt.Sprintf("Unsupported proxy rule type %q.", plan.Type.ValueString()))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error creating proxy rule", common.FormatError(err))
		return
	}

	// The create response can contain placeholder values for certificate fields
	// (for example, renewAt may be "datetime"). Read the rule back so the initial
	// state matches subsequent refreshes and imports.
	rule, err = proxyClient.GetRule(rule.Id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading created proxy rule", common.FormatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(rule, &plan)
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
	proxyClient := appwrite.NewProxy(r.clients.ClientForProject(projectID))

	rule, err := proxyClient.GetRule(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading proxy rule", common.FormatError(err))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	r.mapToState(rule, &state)
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
	proxyClient := appwrite.NewProxy(r.clients.ClientForProject(projectID))

	_, err = proxyClient.DeleteRule(state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting proxy rule", common.FormatError(err))
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

func (r *ruleResource) mapToState(rule *models.ProxyRule, model *ruleResourceModel) {
	model.ID = types.StringValue(rule.Id)
	model.Domain = types.StringValue(rule.Domain)
	model.Type = types.StringValue(rule.DeploymentResourceType)
	model.ResourceID = types.StringValue(rule.DeploymentResourceId)
	model.Status = types.StringValue(rule.Status)
	model.Logs = types.StringValue(rule.Logs)
	model.RenewAt = types.StringValue(rule.RenewAt)
	model.CreatedAt = types.StringValue(rule.CreatedAt)
	model.UpdatedAt = types.StringValue(rule.UpdatedAt)
	if rule.DeploymentVcsProviderBranch != "" {
		model.Branch = types.StringValue(rule.DeploymentVcsProviderBranch)
	} else if !model.Branch.IsNull() {
		model.Branch = types.StringValue("")
	}
}
