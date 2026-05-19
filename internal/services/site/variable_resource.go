package site

import (
	"context"
	"fmt"
	"strings"

	"github.com/appwrite/sdk-for-go/v4/appwrite"
	"github.com/appwrite/sdk-for-go/v4/id"
	"github.com/appwrite/sdk-for-go/v4/models"
	"github.com/appwrite/sdk-for-go/v4/sites"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &variableResource{}
	_ resource.ResourceWithConfigure   = &variableResource{}
	_ resource.ResourceWithImportState = &variableResource{}
)

type variableResource struct {
	clients *common.AppwriteClients
}

type variableResourceModel struct {
	ID        types.String `tfsdk:"id"`
	SiteID    types.String `tfsdk:"site_id"`
	Key       types.String `tfsdk:"key"`
	Value     types.String `tfsdk:"value"`
	Secret    types.Bool   `tfsdk:"secret"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
	ProjectID types.String `tfsdk:"project_id"`
}

func NewVariableResource() resource.Resource {
	return &variableResource{}
}

func (r *variableResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site_variable"
}

func (r *variableResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Appwrite site environment variable.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The variable ID.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"site_id": schema.StringAttribute{
				Description:   "The site ID this variable belongs to.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"key": schema.StringAttribute{
				Description: "The variable key (name).",
				Required:    true,
			},
			"value": schema.StringAttribute{
				Description: "The variable value.",
				Required:    true,
				Sensitive:   true,
			},
			"secret": schema.BoolAttribute{
				Description: "Whether the variable is secret. Secret variables can only be updated or deleted, never read.",
				Optional:    true,
			},
			"created_at": schema.StringAttribute{
				Description:   "The variable creation timestamp in ISO 8601 format.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Description: "The variable last update timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"project_id": common.ProjectIDAttribute(),
		},
	}
}

func (r *variableResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *variableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan variableResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	sitesClient := appwrite.NewSites(r.clients.ClientForProject(projectID))

	var createOpts []sites.CreateVariableOption
	if !plan.Secret.IsNull() {
		createOpts = append(createOpts, sitesClient.WithCreateVariableSecret(plan.Secret.ValueBool()))
	}

	variable, err := sitesClient.CreateVariable(
		plan.SiteID.ValueString(),
		id.Unique(),
		plan.Key.ValueString(),
		plan.Value.ValueString(),
		createOpts...,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error creating site variable", common.FormatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(variable, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *variableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state variableResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	sitesClient := appwrite.NewSites(r.clients.ClientForProject(projectID))

	variable, err := sitesClient.GetVariable(state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading site variable", common.FormatError(err))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	r.mapToState(variable, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *variableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan variableResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	sitesClient := appwrite.NewSites(r.clients.ClientForProject(projectID))

	updateOpts := []sites.UpdateVariableOption{
		sitesClient.WithUpdateVariableKey(plan.Key.ValueString()),
		sitesClient.WithUpdateVariableValue(plan.Value.ValueString()),
	}
	if !plan.Secret.IsNull() {
		updateOpts = append(updateOpts, sitesClient.WithUpdateVariableSecret(plan.Secret.ValueBool()))
	}

	variable, err := sitesClient.UpdateVariable(
		plan.SiteID.ValueString(),
		plan.ID.ValueString(),
		updateOpts...,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error updating site variable", common.FormatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(variable, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *variableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state variableResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	sitesClient := appwrite.NewSites(r.clients.ClientForProject(projectID))

	_, err = sitesClient.DeleteVariable(state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting site variable", common.FormatError(err))
	}
}

func (r *variableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: site_id/variable_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *variableResource) mapToState(variable *models.Variable, model *variableResourceModel) {
	model.ID = types.StringValue(variable.Id)
	model.Key = types.StringValue(variable.Key)
	model.CreatedAt = types.StringValue(variable.CreatedAt)
	model.UpdatedAt = types.StringValue(variable.UpdatedAt)
	if variable.Value != "" {
		model.Value = types.StringValue(variable.Value)
	}
}
