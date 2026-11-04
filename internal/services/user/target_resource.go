package user

import (
	"context"
	"fmt"
	"strings"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/id"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/sdk-for-go/v7/users"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

var (
	_ resource.Resource                   = &targetResource{}
	_ resource.ResourceWithConfigure      = &targetResource{}
	_ resource.ResourceWithImportState    = &targetResource{}
	_ resource.ResourceWithValidateConfig = &targetResource{}
)

type targetResource struct {
	clients *common.AppwriteClients
}

type targetResourceModel struct {
	ProjectID    types.String `tfsdk:"project_id"`
	ID           types.String `tfsdk:"id"`
	UserID       types.String `tfsdk:"user_id"`
	ProviderType types.String `tfsdk:"provider_type"`
	Identifier   types.String `tfsdk:"identifier"`
	ProviderID   types.String `tfsdk:"provider_id"`
	Name         types.String `tfsdk:"name"`
	Expired      types.Bool   `tfsdk:"expired"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func NewTargetResource() resource.Resource {
	return &targetResource{}
}

func (r *targetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auth_user_target"
}

// requiresReplaceIfRemoved plans a replacement when an argument that was set is
// removed from the configuration. Appwrite's update route treats an omitted or
// empty value as "leave unchanged", so a name or provider, once set, cannot be
// cleared in place.
func requiresReplaceIfRemoved() planmodifier.String {
	return stringplanmodifier.RequiresReplaceIf(
		func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
			resp.RequiresReplace = !req.StateValue.IsNull() && req.PlanValue.IsNull()
		},
		"Removing this argument forces a new resource to be created, because Appwrite cannot clear it.",
		"Removing this argument forces a new resource to be created, because Appwrite cannot clear it.",
	)
}

func (r *targetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	idValidators := []validator.String{
		stringvalidator.UTF8LengthBetween(1, maxTargetIDLength),
		stringvalidator.RegexMatches(targetIDPattern,
			"must contain only letters, digits, periods, hyphens and underscores, and must not start with a period, hyphen or underscore"),
	}

	resp.Schema = schema.Schema{
		Description: "Manages a messaging target of an Appwrite user: an email address, phone number or push " +
			"token that messages can be sent to, and that appwrite_messaging_subscriber can subscribe to a topic.",
		Attributes: map[string]schema.Attribute{
			"project_id": common.ProjectIDAttribute(),
			"id": schema.StringAttribute{
				Description:   "The target ID. Generated when not set. " + common.ForcesReplacementNote,
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
				Validators:    idValidators,
			},
			"user_id": schema.StringAttribute{
				Description:   "The ID of the user the target belongs to. " + common.ForcesReplacementNote,
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"provider_type": schema.StringAttribute{
				Description:   "The kind of target: `email`, `sms` or `push`. " + common.ForcesReplacementNote,
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.OneOf(providerTypeEmail, providerTypeSMS, providerTypePush),
				},
			},
			"identifier": schema.StringAttribute{
				Description: "Where messages are sent: an email address for `email`, a phone number in E.164 " +
					"format for `sms`, or a device token for `push`. Must be unique across the project, " +
					"including the email addresses and phone numbers of its users.",
				Required:   true,
				Validators: []validator.String{stringvalidator.UTF8LengthBetween(1, maxTargetIdentifierLength)},
			},
			"provider_id": schema.StringAttribute{
				Description: "The ID of the messaging provider to send through, which must be of the same type as " +
					"the target. When not set, Appwrite uses the project's first provider of that type. " +
					"Removing it forces a new resource to be created, because Appwrite cannot clear it.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{requiresReplaceIfRemoved()},
				Validators:    idValidators,
			},
			"name": schema.StringAttribute{
				Description: "A label for the target, such as the device a push token belongs to. " +
					"Removing it forces a new resource to be created, because Appwrite cannot clear it.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{requiresReplaceIfRemoved()},
				Validators:    []validator.String{stringvalidator.UTF8LengthBetween(1, maxTargetNameLength)},
			},
			"expired": schema.BoolAttribute{
				Description: "Whether a push provider has reported the device token as expired. Changing the " +
					"identifier clears it.",
				Computed: true,
			},
			"created_at": schema.StringAttribute{
				Description:   "The target creation timestamp in ISO 8601 format.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Description:   "The target last update timestamp in ISO 8601 format.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{common.UseStateForUnknownUnlessUpdating()},
			},
		},
	}
}

// ValidateConfig rejects an identifier the provider type cannot take, which
// Appwrite would otherwise only report mid-apply.
func (r *targetResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config targetResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Either may come from another resource and not be known until apply.
	if config.ProviderType.IsNull() || config.ProviderType.IsUnknown() || config.Identifier.IsNull() || config.Identifier.IsUnknown() {
		return
	}
	resp.Diagnostics.Append(validateTargetIdentifier(config.ProviderType.ValueString(), config.Identifier.ValueString())...)
}

func (r *targetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *targetResource) client(projectID types.String, diags *diag.Diagnostics) (string, *users.Users, bool) {
	resolved, err := common.ResolveProjectID(r.clients, projectID)
	if err != nil {
		diags.AddError("Error resolving project ID", err.Error())
		return "", nil, false
	}
	return resolved, appwrite.NewUsers(r.clients.ClientForProject(resolved)), true
}

func (r *targetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan targetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, client, ok := r.client(plan.ProjectID, &resp.Diagnostics)
	if !ok {
		return
	}

	targetID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		targetID = id.Unique()
	}

	target, err := createTarget(client, plan, targetID)
	if target == nil {
		resp.Diagnostics.AddError("Error creating user target", formatTargetError(err, true))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	mapTargetToState(target, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	// The target exists but its provider was refused. With state recorded and
	// an error returned, Terraform marks the resource tainted and replaces it
	// on the next apply, rather than leaving a target nothing tracks.
	if err != nil {
		resp.Diagnostics.AddError("Error setting user target provider", formatTargetError(err, false))
	}
}

func (r *targetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state targetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, client, ok := r.client(state.ProjectID, &resp.Diagnostics)
	if !ok {
		return
	}

	// A deleted target and a deleted user both come back as 404. Deleting a
	// user deletes its targets.
	target, err := client.GetTarget(state.UserID.ValueString(), state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading user target", formatTargetError(err, false))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	mapTargetToState(target, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *targetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state targetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, client, ok := r.client(plan.ProjectID, &resp.Diagnostics)
	if !ok {
		return
	}

	userID, targetID := state.UserID.ValueString(), state.ID.ValueString()
	opts := targetUpdateOptions(client, state, plan)

	var target *models.Target
	var err error
	if len(opts) > 0 {
		target, err = client.UpdateTarget(userID, targetID, opts...)
	} else {
		// Every argument that reaches an update is one that can change in
		// place, so this is not expected; read rather than send an empty
		// update.
		target, err = client.GetTarget(userID, targetID)
	}
	if err != nil {
		resp.Diagnostics.AddError("Error updating user target", formatTargetError(err, !plan.Identifier.Equal(state.Identifier)))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	mapTargetToState(target, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *targetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state targetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, client, ok := r.client(state.ProjectID, &resp.Diagnostics)
	if !ok {
		return
	}

	_, err := client.DeleteTarget(state.UserID.ValueString(), state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting user target", formatTargetError(err, false))
	}
}

func (r *targetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: user_id/target_id, got: %s", req.ID))
		return
	}
	userID, targetID := parts[0], parts[1]

	_, client, ok := r.client(types.StringNull(), &resp.Diagnostics)
	if !ok {
		return
	}
	resp.Diagnostics.Append(checkTargetImportable(client, userID, targetID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), userID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), targetID)...)
}

// checkTargetImportable refuses the target Appwrite keeps for a user's own
// email address or phone number.
//
// It would import cleanly and then be managed from two places. Changing the
// user's email or phone rewrites the target's identifier, which this resource
// would then plan to change back; destroying this resource deletes it, and
// the user stops receiving messages at their own address until it changes.
// Subscribe that target by its ID instead, without managing it.
func checkTargetImportable(client *users.Users, userID, targetID string) diag.Diagnostics {
	var diags diag.Diagnostics
	user, err := client.Get(userID)
	if err != nil {
		diags.AddError("Error reading user", formatTargetError(err, false))
		return diags
	}
	target := findTarget(user, targetID)
	if target == nil {
		diags.AddError("Cannot import user target", fmt.Sprintf("User %q has no target %q.", userID, targetID))
		return diags
	}
	if isContactTarget(user, target) {
		diags.AddError("Cannot import a user's own contact target",
			fmt.Sprintf("Target %q is the %s target Appwrite keeps for user %q's own %s. Appwrite updates it when the "+
				"user's %s changes and deletes it when that is removed, so Terraform cannot manage it as a separate "+
				"resource. Pass its ID to appwrite_messaging_subscriber directly instead.",
				targetID, target.ProviderType, userID, contactKind(target.ProviderType), contactKind(target.ProviderType)))
	}
	return diags
}

func contactKind(providerType string) string {
	if providerType == providerTypeSMS {
		return "phone number"
	}
	return "email address"
}
