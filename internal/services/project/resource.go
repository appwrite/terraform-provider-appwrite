package project

import (
	"context"
	"fmt"
	"strings"

	"github.com/appwrite/sdk-for-go/v6/appwrite"
	"github.com/appwrite/sdk-for-go/v6/id"
	"github.com/appwrite/sdk-for-go/v6/models"
	"github.com/appwrite/sdk-for-go/v6/organization"
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
	_ resource.Resource                = &projectResource{}
	_ resource.ResourceWithConfigure   = &projectResource{}
	_ resource.ResourceWithImportState = &projectResource{}
)

type projectResource struct {
	clients *common.AppwriteClients
}

type projectResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Region         types.String `tfsdk:"region"`
	OrganizationID types.String `tfsdk:"organization_id"`
	TeamID         types.String `tfsdk:"team_id"`
	Status         types.String `tfsdk:"status"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func NewProjectResource() resource.Resource {
	return &projectResource{}
}

func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Appwrite project within an organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The project ID.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The project name.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1), stringvalidator.LengthAtMost(128)},
			},
			"region": schema.StringAttribute{
				Description: "The region where the project is hosted. Defaults to the server's configured region.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": common.OrganizationIDAttribute(),
			"team_id": schema.StringAttribute{
				Description: "The ID of the team that owns the project.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The project status.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The project creation timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The project last update timestamp in ISO 8601 format.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					common.UseStateForUnknownUnlessUpdating(),
				},
			},
		},
	}
}

func (r *projectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID, err := common.ResolveOrganizationID(r.clients, plan.OrganizationID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving organization ID", err.Error())
		return
	}
	organizationClient, err := r.client(organizationID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	projectID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		projectID = id.Unique()
	}

	var opts []organization.CreateProjectOption
	if !plan.Region.IsNull() && !plan.Region.IsUnknown() {
		opts = append(opts, organizationClient.WithCreateProjectRegion(plan.Region.ValueString()))
	}

	created, err := organizationClient.CreateProject(projectID, plan.Name.ValueString(), opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error creating project", common.FormatErrorWithAuthGuidance(err, common.OrganizationCredentialGuidance("appwrite_project", "projects.read", "projects.write")))
		return
	}

	plan.OrganizationID = types.StringValue(organizationID)
	r.mapToState(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID, err := common.ResolveOrganizationID(r.clients, state.OrganizationID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving organization ID", err.Error())
		return
	}
	organizationClient, err := r.client(organizationID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	found, err := organizationClient.GetProject(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading project", common.FormatErrorWithAuthGuidance(err, common.OrganizationCredentialGuidance("appwrite_project", "projects.read", "projects.write")))
		return
	}

	state.OrganizationID = types.StringValue(organizationID)
	r.mapToState(found, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID, err := common.ResolveOrganizationID(r.clients, plan.OrganizationID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving organization ID", err.Error())
		return
	}
	organizationClient, err := r.client(organizationID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	updated, err := organizationClient.UpdateProject(plan.ID.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error updating project", common.FormatErrorWithAuthGuidance(err, common.OrganizationCredentialGuidance("appwrite_project", "projects.read", "projects.write")))
		return
	}

	plan.OrganizationID = types.StringValue(organizationID)
	r.mapToState(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID, err := common.ResolveOrganizationID(r.clients, state.OrganizationID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving organization ID", err.Error())
		return
	}
	organizationClient, err := r.client(organizationID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	_, err = organizationClient.DeleteProject(state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting project", common.FormatErrorWithAuthGuidance(err, common.OrganizationCredentialGuidance("appwrite_project", "projects.read", "projects.write")))
	}
}

func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	switch len(parts) {
	case 1:
		if parts[0] == "" {
			resp.Diagnostics.AddError("Invalid import ID", "Expected project_id or organization_id/project_id")
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[0])...)
	case 2:
		if parts[0] == "" || parts[1] == "" {
			resp.Diagnostics.AddError("Invalid import ID", "Expected project_id or organization_id/project_id")
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	default:
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected project_id or organization_id/project_id, got: %s", req.ID))
	}
}

func (r *projectResource) client(organizationID string) (*organization.Organization, error) {
	if err := common.ValidateOrganizationCredential(r.clients, "appwrite_project", "projects.read", "projects.write"); err != nil {
		return nil, err
	}
	return appwrite.NewOrganization(r.clients.ClientForOrganization(organizationID)), nil
}

func (r *projectResource) mapToState(project *models.Project, model *projectResourceModel) {
	model.ID = types.StringValue(project.Id)
	model.Name = types.StringValue(project.Name)
	model.Region = types.StringValue(project.Region)
	model.TeamID = types.StringValue(project.TeamId)
	model.Status = types.StringValue(project.Status)
	model.CreatedAt = types.StringValue(project.CreatedAt)
	model.UpdatedAt = types.StringValue(project.UpdatedAt)
}
