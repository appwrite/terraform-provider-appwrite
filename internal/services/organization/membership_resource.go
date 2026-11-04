package organization

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

var (
	_ resource.Resource                     = &membershipResource{}
	_ resource.ResourceWithConfigure        = &membershipResource{}
	_ resource.ResourceWithConfigValidators = &membershipResource{}
	_ resource.ResourceWithImportState      = &membershipResource{}
)

type membershipResource struct {
	clients *common.AppwriteClients
}

type membershipResourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Email          types.String `tfsdk:"email"`
	UserID         types.String `tfsdk:"user_id"`
	Name           types.String `tfsdk:"name"`
	Roles          types.Set    `tfsdk:"roles"`
	UserName       types.String `tfsdk:"user_name"`
	UserEmail      types.String `tfsdk:"user_email"`
	Confirm        types.Bool   `tfsdk:"confirm"`
	JoinedAt       types.String `tfsdk:"joined_at"`
}

func NewMembershipResource() resource.Resource {
	return &membershipResource{}
}

func (r *membershipResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_membership"
}

func (r *membershipResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	useState := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	resp.Schema = schema.Schema{
		Description: "Adds a member to an Appwrite organization and manages their roles.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The membership ID.",
				Computed:      true,
				PlanModifiers: useState,
			},
			"organization_id": common.OrganizationIDAttribute(),
			"email": schema.StringAttribute{
				Description: "The email address of the member to add. When no Appwrite account uses it, Appwrite " +
					"creates one. Kept as written; see user_email for the account's current address. Exactly one of " +
					"email and user_id must be set. " + common.ForcesReplacementNote + " A change only in letter case " +
					"does not.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplaceIf(
						emailChangeRequiresReplace,
						"Changing the email address replaces the membership, unless only its letter case changes.",
						"Changing the email address replaces the membership, unless only its letter case changes.",
					),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(emailPattern, "must be an email address"),
				},
			},
			"user_id": schema.StringAttribute{
				Description: "The ID of an existing Appwrite account to add. Exactly one of email and user_id must be set. " +
					common.ForcesReplacementNote,
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtMost(maxUserIDLen),
					stringvalidator.RegexMatches(keyPattern, "must contain only letters, digits, periods, hyphens and underscores, and must not start with a period, hyphen or underscore"),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name to give the account Appwrite creates when no account uses email. Ignored " +
					"when the account already exists, and not read back. Changing it later updates state only.",
				Optional:   true,
				Validators: []validator.String{stringvalidator.LengthBetween(1, maxNameLength)},
			},
			"roles": schema.SetAttribute{
				Description: "The member's roles in the organization: owner, developer, editor, analyst or billing, " +
					"or a project-specific role of the form project-{project_id}-{role} on plans that support them. " +
					"Updated in place.",
				ElementType: types.StringType,
				Required:    true,
				Validators: []validator.Set{
					setvalidator.SizeBetween(1, maxRoles),
					setvalidator.ValueStringsAre(
						stringvalidator.LengthAtMost(maxRoleLength),
						stringvalidator.RegexMatches(keyPattern, "must contain only letters, digits, periods, hyphens and underscores, and must not start with a period, hyphen or underscore"),
					),
				},
			},
			"user_name": schema.StringAttribute{
				Description:   "The member account's name.",
				Computed:      true,
				PlanModifiers: useState,
			},
			"user_email": schema.StringAttribute{
				Description:   "The member account's current email address.",
				Computed:      true,
				PlanModifiers: useState,
			},
			"confirm": schema.BoolAttribute{
				Description: "Whether the member has joined. Members added by Terraform join immediately; a " +
					"membership imported while its invitation is pending is false until it is accepted.",
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"joined_at": schema.StringAttribute{
				Description:   "When the member joined, in ISO 8601 format. Empty while an invitation is pending.",
				Computed:      true,
				PlanModifiers: useState,
			},
		},
	}
}

// emailChangeRequiresReplace replaces the membership when the email address
// changes, except in letter case. Appwrite matches the address to an account
// without regard to case, so a membership imported with the stored lowercase
// address and configured as Jane@Example.com is the same membership.
func emailChangeRequiresReplace(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
	if req.PlanValue.IsUnknown() || req.StateValue.IsUnknown() {
		resp.RequiresReplace = true
		return
	}
	resp.RequiresReplace = !strings.EqualFold(req.PlanValue.ValueString(), req.StateValue.ValueString())
}

func (r *membershipResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("email"),
			path.MatchRoot("user_id"),
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

func (r *membershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan membershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	roles, diags := planRoles(ctx, plan.Roles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID, err := common.ResolveOrganizationID(r.clients, plan.OrganizationID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving organization ID", err.Error())
		return
	}
	client, err := membershipClient(r.clients, organizationID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	created, err := createMembership(client, plan, roles)
	if err != nil {
		resp.Diagnostics.AddError("Error creating organization membership", membershipError(err))
		return
	}

	plan.OrganizationID = types.StringValue(organizationID)
	resp.Diagnostics.Append(mapMembershipToState(ctx, created, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *membershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state membershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID, err := common.ResolveOrganizationID(r.clients, state.OrganizationID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving organization ID", err.Error())
		return
	}
	client, err := membershipClient(r.clients, organizationID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	// A membership of another organization is a 404 too, so a wrong
	// organization_id on import reads as a missing membership.
	found, err := client.GetMembership(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading organization membership", membershipError(err))
		return
	}

	state.OrganizationID = types.StringValue(organizationID)
	resp.Diagnostics.Append(mapMembershipToState(ctx, found, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update changes the roles. A change to email letter case or to name reaches
// here too, and only updates state.
func (r *membershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state membershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	roles, diags := planRoles(ctx, plan.Roles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID, err := common.ResolveOrganizationID(r.clients, plan.OrganizationID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving organization ID", err.Error())
		return
	}
	client, err := membershipClient(r.clients, organizationID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	current, err := updateMembership(client, state.ID.ValueString(), roles, !plan.Roles.Equal(state.Roles))
	if err != nil {
		resp.Diagnostics.AddError("Error updating organization membership", membershipError(err))
		return
	}

	plan.ID = state.ID
	plan.OrganizationID = types.StringValue(organizationID)
	resp.Diagnostics.Append(mapMembershipToState(ctx, current, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *membershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state membershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID, err := common.ResolveOrganizationID(r.clients, state.OrganizationID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving organization ID", err.Error())
		return
	}
	client, err := membershipClient(r.clients, organizationID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	// The server refuses to remove the last confirmed owner, and does so under
	// a lock on the organization. Checking here first would only race it.
	_, err = client.DeleteMembership(state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting organization membership", membershipError(err))
	}
}

func (r *membershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	const expected = "Expected membership_id or organization_id/membership_id"
	parts := strings.Split(req.ID, "/")
	switch len(parts) {
	case 1:
		if parts[0] == "" {
			resp.Diagnostics.AddError("Invalid import ID", expected)
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[0])...)
	case 2:
		if parts[0] == "" || parts[1] == "" {
			resp.Diagnostics.AddError("Invalid import ID", expected)
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	default:
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("%s, got: %s", expected, req.ID))
	}
}
