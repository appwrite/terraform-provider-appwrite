package team

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/id"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/sdk-for-go/v7/teams"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &teamResource{}
	_ resource.ResourceWithConfigure   = &teamResource{}
	_ resource.ResourceWithImportState = &teamResource{}
	_ resource.ResourceWithModifyPlan  = &teamResource{}
)

type teamResource struct {
	clients *common.AppwriteClients
}

type teamResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Roles     types.List   `tfsdk:"roles"`
	Prefs     types.String `tfsdk:"prefs"`
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
				Description:   "The team ID. Changing this forces a new resource to be created.",
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
			"prefs": schema.StringAttribute{
				Description: prefsDescription + " When omitted, Terraform leaves the preferences as they are and " +
					"reports what the server holds. Setting it replaces the whole object. Appwrite does not accept " +
					"an empty object, so preferences cannot be cleared once set: \"{}\" is rejected at plan time " +
					"for a team that has preferences.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{prefsValidator{}},
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

// ModifyPlan refuses, at plan time, to clear preferences that are already set.
// Appwrite rejects an empty prefs object, so the apply would otherwise fail
// part-way, after any other change in the same plan had been made.
func (r *teamResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var planned, current types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("prefs"), &planned)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("prefs"), &current)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if planned.IsNull() || planned.IsUnknown() || current.IsNull() || current.IsUnknown() {
		return
	}
	if prefsEqual(planned.ValueString(), emptyPrefs) && !prefsEqual(current.ValueString(), emptyPrefs) {
		resp.Diagnostics.AddAttributeError(path.Root("prefs"), "Team preferences cannot be cleared",
			"Appwrite does not accept an empty preferences object, so preferences cannot be removed once set. "+
				"Set the keys you want to keep, or remove prefs from the configuration to stop managing them.")
	}
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

	desired := plan.Prefs
	plan.ProjectID = types.StringValue(projectID)
	plan.Prefs = types.StringNull()
	resp.Diagnostics.Append(mapTeamToState(team, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A new team starts with empty preferences, so only a configuration
	// asking for something else needs a second call. The team is saved
	// first: if setting its preferences fails, it is tainted and replaced on
	// the next apply rather than left on the server with nothing in state
	// pointing at it.
	if !desired.IsNull() && !desired.IsUnknown() && !prefsEqual(desired.ValueString(), plan.Prefs.ValueString()) {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
		team, err = updatePrefs(teamsClient, team.Id, desired.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error setting team preferences", common.FormatError(err))
			return
		}
	}
	if !desired.IsNull() && !desired.IsUnknown() {
		plan.Prefs = desired
	}
	resp.Diagnostics.Append(mapTeamToState(team, &plan)...)
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
	resp.Diagnostics.Append(mapTeamToState(team, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *teamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, current teamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &current)...)
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

	// Preferences are written only when they differ from what the server
	// holds. An omitted prefs plans as the prior value, which is what keeps a
	// configuration that never mentions them from touching them.
	if !plan.Prefs.IsNull() && !plan.Prefs.IsUnknown() && !prefsEqual(plan.Prefs.ValueString(), current.Prefs.ValueString()) {
		team, err = updatePrefs(teamsClient, team.Id, plan.Prefs.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error updating team preferences", common.FormatError(err))
			return
		}
	}

	plan.ProjectID = types.StringValue(projectID)
	resp.Diagnostics.Append(mapTeamToState(team, &plan)...)
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

// updatePrefs replaces a team's preferences and returns the team as it then
// stands. The prefs endpoint answers with the preferences alone, so the team
// is read back for its new update timestamp.
func updatePrefs(client *teams.Teams, teamID, prefs string) (*models.Team, error) {
	object, err := common.DecodeJSONObject(prefs)
	if err != nil {
		return nil, fmt.Errorf("decoding prefs: %w", err)
	}
	if _, err := client.UpdatePrefs(teamID, object); err != nil {
		return nil, err
	}
	return client.Get(teamID)
}

// mapTeamToState copies a team into the resource model. Preferences already in
// the model are kept when they hold the same object, so the configured
// spelling survives a refresh.
func mapTeamToState(team *models.Team, model *teamResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	model.ID = types.StringValue(team.Id)
	model.Name = types.StringValue(team.Name)
	model.CreatedAt = types.StringValue(team.CreatedAt)
	model.UpdatedAt = types.StringValue(team.UpdatedAt)

	prefs, err := teamPrefs(team, model.Prefs)
	if err != nil {
		diags.AddError("Error reading team preferences", err.Error())
		return diags
	}
	model.Prefs = prefs
	return diags
}

// teamPrefs returns the team's preferences as they belong in state, given the
// value already there.
func teamPrefs(team *models.Team, prior types.String) (types.String, error) {
	raw, err := rawPrefs(team.Decode)
	if err != nil {
		return types.StringNull(), err
	}
	return prefsForState(prior, raw)
}
