package team

import (
	"context"
	"fmt"
	"strings"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/sdk-for-go/v7/query"
	"github.com/appwrite/sdk-for-go/v7/teams"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                     = &membershipResource{}
	_ resource.ResourceWithConfigure        = &membershipResource{}
	_ resource.ResourceWithImportState      = &membershipResource{}
	_ resource.ResourceWithConfigValidators = &membershipResource{}
)

type membershipResource struct {
	clients *common.AppwriteClients
}

type membershipResourceModel struct {
	ID        types.String `tfsdk:"id"`
	TeamID    types.String `tfsdk:"team_id"`
	UserID    types.String `tfsdk:"user_id"`
	Email     types.String `tfsdk:"email"`
	Phone     types.String `tfsdk:"phone"`
	Name      types.String `tfsdk:"name"`
	Roles     types.Set    `tfsdk:"roles"`
	Confirm   types.Bool   `tfsdk:"confirm"`
	JoinedAt  types.String `tfsdk:"joined_at"`
	UserName  types.String `tfsdk:"user_name"`
	UserEmail types.String `tfsdk:"user_email"`
	UserPhone types.String `tfsdk:"user_phone"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
	ProjectID types.String `tfsdk:"project_id"`
}

func NewMembershipResource() resource.Resource {
	return &membershipResource{}
}

func (r *membershipResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auth_team_membership"
}

func (r *membershipResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	adoptable := []planmodifier.String{requiresReplaceUnlessAdopting()}
	unchanging := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}

	resp.Schema = schema.Schema{
		Description: "Manages a user's membership of an Appwrite team. Because the provider authenticates with an " +
			"API key, Appwrite confirms the membership immediately and sends no invitation. Identify the member by " +
			"exactly one of user_id, email or phone. Given an email address or phone number that belongs to no " +
			"user, Appwrite creates one; destroying the membership does not delete that user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The membership ID, assigned by Appwrite.",
				Computed:      true,
				PlanModifiers: unchanging,
			},
			"team_id": schema.StringAttribute{
				Description:   "The ID of the team. " + common.ForcesReplacementNote,
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"user_id": schema.StringAttribute{
				Description: "The ID of an existing user to add to the team. When the member is given by email or " +
					"phone instead, this is the ID of the user Appwrite matched or created. " + common.ForcesReplacementNote,
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"email": schema.StringAttribute{
				Description: "The email address of the member. Appwrite adds the user with this address, or creates " +
					"one if there is none. Appwrite matches it in lowercase. Not read back: a later change to the " +
					"user's address shows in user_email, not here. " + common.ForcesReplacementNote,
				Optional:      true,
				PlanModifiers: adoptable,
			},
			"phone": schema.StringAttribute{
				Description: "The phone number of the member, in E.164 format such as +16175551212. Appwrite adds the " +
					"user with this number, or creates one if there is none. Not read back: a later change to the " +
					"user's number shows in user_phone, not here. " + common.ForcesReplacementNote,
				Optional:      true,
				PlanModifiers: adoptable,
			},
			"name": schema.StringAttribute{
				Description: "The name to give the user Appwrite creates when email or phone matches no user. Ignored " +
					"when the user already exists. Defaults to the email address, or no name for a phone number. " +
					common.ForcesReplacementNote,
				Optional:      true,
				PlanModifiers: adoptable,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(128),
					stringvalidator.ConflictsWith(path.MatchRoot("user_id")),
				},
			},
			"roles": schema.SetAttribute{
				Description: "The member's roles in the team, which permissions refer to as team:TEAM_ID/ROLE. " +
					"May be empty. Order does not matter. Updated in place.",
				Required:    true,
				ElementType: types.StringType,
				Validators:  roleValidators(),
			},
			"confirm": schema.BoolAttribute{
				Description: "Whether the user has joined the team. Always true for a membership created by this " +
					"resource; an imported membership may still be a pending invitation.",
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"joined_at": schema.StringAttribute{
				Description:   "When the user joined the team, in ISO 8601 format. Empty for a pending invitation.",
				Computed:      true,
				PlanModifiers: unchanging,
			},
			"user_name": schema.StringAttribute{
				Description:   "The member's current name.",
				Computed:      true,
				PlanModifiers: unchanging,
			},
			"user_email": schema.StringAttribute{
				Description:   "The member's current email address.",
				Computed:      true,
				PlanModifiers: unchanging,
			},
			"user_phone": schema.StringAttribute{
				Description:   "The member's current phone number.",
				Computed:      true,
				PlanModifiers: unchanging,
			},
			"created_at": schema.StringAttribute{
				Description:   "The membership creation timestamp in ISO 8601 format.",
				Computed:      true,
				PlanModifiers: unchanging,
			},
			"updated_at": schema.StringAttribute{
				Description: "The membership last update timestamp in ISO 8601 format.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					common.UseStateForUnknownUnlessUpdating(),
				},
			},
			"project_id": common.ProjectIDAttribute(),
		},
	}
}

func (r *membershipResource) ConfigValidators(context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("user_id"),
			path.MatchRoot("email"),
			path.MatchRoot("phone"),
		),
	}
}

func (r *membershipResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *membershipResource) teamsClient(projectID types.String) (*teams.Teams, string, error) {
	resolved, err := common.ResolveProjectID(r.clients, projectID)
	if err != nil {
		return nil, "", err
	}
	return appwrite.NewTeams(r.clients.ClientForProject(resolved)), resolved, nil
}

func (r *membershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan membershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, projectID, err := r.teamsClient(plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}

	var roles []string
	resp.Diagnostics.Append(plan.Roles.ElementsAs(ctx, &roles, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	membership, err := createMembership(client, plan, roles)
	if err != nil {
		resp.Diagnostics.AddError("Error creating team membership", common.FormatError(err)+alreadyMemberHint(client, plan, err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	resp.Diagnostics.Append(mapMembershipToState(ctx, membership, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The create response leaves out the member's phone number, which only
	// the read endpoint fills in. Read it back so that state after create
	// matches state after refresh.
	full, err := client.GetMembership(membership.TeamId, membership.Id)
	if err != nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError("Error reading team membership after create", common.FormatError(err))
		return
	}
	resp.Diagnostics.Append(mapMembershipToState(ctx, full, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *membershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state membershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, projectID, err := r.teamsClient(state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}

	// A membership disappears with its team or its user, and Appwrite reports
	// a membership of another team as not found too, so every 404 means the
	// membership is gone.
	membership, err := client.GetMembership(state.TeamID.ValueString(), state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading team membership", common.FormatError(err))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	resp.Diagnostics.Append(mapMembershipToState(ctx, membership, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *membershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, current membershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &current)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, projectID, err := r.teamsClient(plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}

	teamID, membershipID := plan.TeamID.ValueString(), plan.ID.ValueString()
	membership, err := client.GetMembership(teamID, membershipID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading team membership", common.FormatError(err))
		return
	}

	// Setting email or phone on an imported membership adopts it rather than
	// replacing it, so the value has to describe the member it already has.
	if msg := adoptionMismatch(plan, current, membership); msg != "" {
		resp.Diagnostics.AddError("Configuration does not match the imported membership", msg)
		return
	}

	if !plan.Roles.Equal(current.Roles) {
		var roles []string
		resp.Diagnostics.Append(plan.Roles.ElementsAs(ctx, &roles, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if _, err := client.UpdateMembership(teamID, membershipID, roles); err != nil {
			resp.Diagnostics.AddError("Error updating team membership roles", common.FormatError(err))
			return
		}
		// The update response leaves out the phone number, as create does.
		membership, err = client.GetMembership(teamID, membershipID)
		if err != nil {
			resp.Diagnostics.AddError("Error reading team membership after update", common.FormatError(err))
			return
		}
	}

	// Whatever was being adopted now is, so the next change to these
	// arguments replaces the membership like any other.
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, importedKey, nil)...)

	plan.ProjectID = types.StringValue(projectID)
	resp.Diagnostics.Append(mapMembershipToState(ctx, membership, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *membershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state membershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, _, err := r.teamsClient(state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}

	_, err = client.DeleteMembership(state.TeamID.ValueString(), state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting team membership", common.FormatError(err))
	}
}

func (r *membershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: team_id/membership_id, got: %s", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, importedKey, []byte("true"))...)
}

// createMembership adds the member the plan identifies to the team.
//
// Exactly one of user_id, email and phone is set; the config validators
// guarantee it. url is never sent: Appwrite uses it only for the invitation
// link it emails when a client creates a membership, and with an API key it
// sends no invitation.
func createMembership(client *teams.Teams, plan membershipResourceModel, roles []string) (*models.Membership, error) {
	var opts []teams.CreateMembershipOption
	if v := plan.UserID; !v.IsNull() && !v.IsUnknown() {
		opts = append(opts, client.WithCreateMembershipUserId(v.ValueString()))
	}
	if v := plan.Email; !v.IsNull() && !v.IsUnknown() {
		opts = append(opts, client.WithCreateMembershipEmail(v.ValueString()))
	}
	if v := plan.Phone; !v.IsNull() && !v.IsUnknown() {
		opts = append(opts, client.WithCreateMembershipPhone(v.ValueString()))
	}
	if v := plan.Name; !v.IsNull() && !v.IsUnknown() {
		opts = append(opts, client.WithCreateMembershipName(v.ValueString()))
	}
	if roles == nil {
		// An empty set is a valid membership with no roles; nil would be
		// encoded as null, which the server rejects.
		roles = []string{}
	}
	membership, err := client.CreateMembership(plan.TeamID.ValueString(), roles, opts...)
	if err != nil {
		return nil, err
	}
	// A pending invitation for the same user is confirmed and returned rather
	// than refused, with the roles it was sent with. Set the configured ones,
	// or state would disagree with the plan.
	if !sameRoles(membership.Roles, roles) {
		return client.UpdateMembership(membership.TeamId, membership.Id, roles)
	}
	return membership, nil
}

// alreadyMemberHint explains a create refused because the user already belongs
// to the team, which is what happens when a membership Terraform does not
// manage exists. It names the import ID when the existing membership can be
// found.
func alreadyMemberHint(client *teams.Teams, plan membershipResourceModel, err error) string {
	if common.ErrorType(err) != "membership_already_confirmed" {
		return ""
	}
	teamID := plan.TeamID.ValueString()
	hint := fmt.Sprintf("\n\nThe user is already a member of team %q. To manage that membership, import it with the ID team_id/membership_id.", teamID)
	if plan.UserID.IsNull() || plan.UserID.IsUnknown() {
		return hint
	}
	list, listErr := client.ListMemberships(teamID, client.WithListMembershipsQueries([]string{
		query.Equal("userId", plan.UserID.ValueString()),
	}))
	if listErr != nil || len(list.Memberships) != 1 {
		return hint
	}
	return fmt.Sprintf("\n\nThe user is already a member of team %q. To manage that membership, import it with the ID %s/%s.",
		teamID, teamID, list.Memberships[0].Id)
}

// adoptionMismatch reports an email or phone being adopted onto an imported
// membership that belongs to someone else. Appwrite stores email addresses in
// lowercase, so those compare without case.
func adoptionMismatch(plan, current membershipResourceModel, membership *models.Membership) string {
	if current.Email.IsNull() && !plan.Email.IsNull() && !strings.EqualFold(plan.Email.ValueString(), membership.UserEmail) {
		return fmt.Sprintf("Membership %s belongs to user %s, whose email address is %q, not %q. "+
			"Correct email, or remove the membership from state and let Terraform create one for that address.",
			membership.Id, membership.UserId, membership.UserEmail, plan.Email.ValueString())
	}
	if current.Phone.IsNull() && !plan.Phone.IsNull() && plan.Phone.ValueString() != membership.UserPhone {
		return fmt.Sprintf("Membership %s belongs to user %s, whose phone number is %q, not %q. "+
			"Correct phone, or remove the membership from state and let Terraform create one for that number.",
			membership.Id, membership.UserId, membership.UserPhone, plan.Phone.ValueString())
	}
	return ""
}

// mapMembershipToState copies what Appwrite reports about a membership into
// the model. email, phone and name are what the membership was created from
// and are left as they are.
func mapMembershipToState(ctx context.Context, membership *models.Membership, model *membershipResourceModel) diag.Diagnostics {
	model.ID = types.StringValue(membership.Id)
	model.TeamID = types.StringValue(membership.TeamId)
	model.UserID = types.StringValue(membership.UserId)
	model.Confirm = types.BoolValue(membership.Confirm)
	model.JoinedAt = types.StringValue(membership.Joined)
	model.UserName = types.StringValue(membership.UserName)
	model.UserEmail = types.StringValue(membership.UserEmail)
	model.UserPhone = types.StringValue(membership.UserPhone)
	model.CreatedAt = types.StringValue(membership.CreatedAt)
	model.UpdatedAt = types.StringValue(membership.UpdatedAt)

	roles, diags := types.SetValueFrom(ctx, types.StringType, uniqueRoles(membership.Roles))
	model.Roles = roles
	return diags
}

// sameRoles reports whether two role lists hold the same roles, ignoring order
// and repeats.
func sameRoles(a, b []string) bool {
	a, b = uniqueRoles(a), uniqueRoles(b)
	if len(a) != len(b) {
		return false
	}
	held := make(map[string]bool, len(a))
	for _, role := range a {
		held[role] = true
	}
	for _, role := range b {
		if !held[role] {
			return false
		}
	}
	return true
}

// uniqueRoles drops repeated roles. Appwrite stores roles as an array and
// keeps duplicates sent by other clients, which a set cannot hold.
func uniqueRoles(roles []string) []string {
	seen := make(map[string]bool, len(roles))
	unique := make([]string, 0, len(roles))
	for _, role := range roles {
		if !seen[role] {
			seen[role] = true
			unique = append(unique, role)
		}
	}
	return unique
}
