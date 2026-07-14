package team

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v6/appwrite"
	"github.com/appwrite/sdk-for-go/v6/id"
	"github.com/appwrite/sdk-for-go/v6/teams"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &teamResource{}
	_ resource.ResourceWithConfigure   = &teamResource{}
	_ resource.ResourceWithImportState = &teamResource{}
)

type teamResource struct {
	clients *common.AppwriteClients
}

type teamResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Roles     types.List   `tfsdk:"roles"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
	ProjectID types.String `tfsdk:"project_id"`
}

func NewTeamResource() resource.Resource {
	return &teamResource{}
}

func (r *teamResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auth_team"
}

func (r *teamResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Appwrite team.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The team ID.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The team name.",
				Required:    true,
			},
			"roles": schema.ListAttribute{
				Description: "Roles for new team members. Defaults to [\"owner\"].",
				Optional:    true,
				ElementType: types.StringType,
			},
			"created_at": schema.StringAttribute{
				Description: "The team creation timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The team last update timestamp in ISO 8601 format.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					common.UseStateForUnknownUnlessUpdating(),
				},
			},
			"project_id": common.ProjectIDAttribute(),
		},
	}
}

func (r *teamResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *teamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan teamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	teamsClient := appwrite.NewTeams(r.clients.ClientForProject(projectID))

	var opts []teams.CreateOption
	if !plan.Roles.IsNull() && !plan.Roles.IsUnknown() {
		var roles []string
		resp.Diagnostics.Append(plan.Roles.ElementsAs(ctx, &roles, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, teamsClient.WithCreateRoles(roles))
	}

	teamID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		teamID = id.Unique()
	}

	team, err := teamsClient.Create(teamID, plan.Name.ValueString(), opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error creating team", common.FormatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	plan.ID = types.StringValue(team.Id)
	plan.Name = types.StringValue(team.Name)
	plan.CreatedAt = types.StringValue(team.CreatedAt)
	plan.UpdatedAt = types.StringValue(team.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *teamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state teamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	teamsClient := appwrite.NewTeams(r.clients.ClientForProject(projectID))

	team, err := teamsClient.Get(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading team", common.FormatError(err))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	state.ID = types.StringValue(team.Id)
	state.Name = types.StringValue(team.Name)
	state.CreatedAt = types.StringValue(team.CreatedAt)
	state.UpdatedAt = types.StringValue(team.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *teamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan teamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	teamsClient := appwrite.NewTeams(r.clients.ClientForProject(projectID))

	team, err := teamsClient.UpdateName(plan.ID.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error updating team", common.FormatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	plan.ID = types.StringValue(team.Id)
	plan.Name = types.StringValue(team.Name)
	plan.CreatedAt = types.StringValue(team.CreatedAt)
	plan.UpdatedAt = types.StringValue(team.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *teamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state teamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	teamsClient := appwrite.NewTeams(r.clients.ClientForProject(projectID))

	_, err = teamsClient.Delete(state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting team", common.FormatError(err))
	}
}

func (r *teamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
