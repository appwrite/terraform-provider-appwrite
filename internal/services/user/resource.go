package user

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v2/appwrite"
	"github.com/appwrite/sdk-for-go/v2/id"
	"github.com/appwrite/sdk-for-go/v2/models"
	"github.com/appwrite/sdk-for-go/v2/users"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &userResource{}
	_ resource.ResourceWithConfigure   = &userResource{}
	_ resource.ResourceWithImportState = &userResource{}
)

type userResource struct {
	clients *common.AppwriteClients
}

type userResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Email             types.String `tfsdk:"email"`
	Phone             types.String `tfsdk:"phone"`
	Password          types.String `tfsdk:"password"`
	Status            types.Bool   `tfsdk:"status"`
	Labels            types.List   `tfsdk:"labels"`
	EmailVerification types.Bool   `tfsdk:"email_verification"`
	PhoneVerification types.Bool   `tfsdk:"phone_verification"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
	ProjectID         types.String `tfsdk:"project_id"`
}

func NewUserResource() resource.Resource {
	return &userResource{}
}

func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auth_user"
}

func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Appwrite user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The user ID.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The user name.",
				Optional:    true,
			},
			"email": schema.StringAttribute{
				Description: "The user email address.",
				Optional:    true,
			},
			"phone": schema.StringAttribute{
				Description: "The user phone number in E.164 format.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "The user password. Write-only, not returned by the API.",
				Optional:    true,
				Sensitive:   true,
			},
			"status": schema.BoolAttribute{
				Description: "Whether the user account is enabled. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"labels": schema.ListAttribute{
				Description: "User labels.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"email_verification": schema.BoolAttribute{
				Description: "Whether the user email is verified.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"phone_verification": schema.BoolAttribute{
				Description: "Whether the user phone is verified.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"created_at": schema.StringAttribute{
				Description: "The user creation timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The user last update timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"project_id": common.ProjectIDAttribute(),
		},
	}
}

func (r *userResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	usersClient := appwrite.NewUsers(r.clients.ClientForProject(projectID))

	var opts []users.CreateOption
	if !plan.Email.IsNull() {
		opts = append(opts, usersClient.WithCreateEmail(plan.Email.ValueString()))
	}
	if !plan.Phone.IsNull() {
		opts = append(opts, usersClient.WithCreatePhone(plan.Phone.ValueString()))
	}
	if !plan.Password.IsNull() {
		opts = append(opts, usersClient.WithCreatePassword(plan.Password.ValueString()))
	}
	if !plan.Name.IsNull() {
		opts = append(opts, usersClient.WithCreateName(plan.Name.ValueString()))
	}

	userID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		userID = id.Unique()
	}

	user, err := usersClient.Create(userID, opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error creating user", common.FormatError(err))
		return
	}

	userID = user.Id

	if !plan.Status.IsNull() && !plan.Status.IsUnknown() && !plan.Status.ValueBool() {
		user, err = usersClient.UpdateStatus(userID, false)
		if err != nil {
			resp.Diagnostics.AddError("Error setting user status", common.FormatError(err))
			return
		}
	}
	if !plan.Labels.IsNull() && !plan.Labels.IsUnknown() {
		var labels []string
		resp.Diagnostics.Append(plan.Labels.ElementsAs(ctx, &labels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		user, err = usersClient.UpdateLabels(userID, labels)
		if err != nil {
			resp.Diagnostics.AddError("Error setting user labels", common.FormatError(err))
			return
		}
	}
	if !plan.EmailVerification.IsNull() && !plan.EmailVerification.IsUnknown() && plan.EmailVerification.ValueBool() {
		user, err = usersClient.UpdateEmailVerification(userID, true)
		if err != nil {
			resp.Diagnostics.AddError("Error setting email verification", common.FormatError(err))
			return
		}
	}
	if !plan.PhoneVerification.IsNull() && !plan.PhoneVerification.IsUnknown() && plan.PhoneVerification.ValueBool() {
		user, err = usersClient.UpdatePhoneVerification(userID, true)
		if err != nil {
			resp.Diagnostics.AddError("Error setting phone verification", common.FormatError(err))
			return
		}
	}

	// Read final state after all updates
	user, err = usersClient.Get(userID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading user after create", common.FormatError(err))
		return
	}

	r.mapToState(ctx, user, &plan, &resp.Diagnostics)
	plan.ProjectID = types.StringValue(projectID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	usersClient := appwrite.NewUsers(r.clients.ClientForProject(projectID))

	user, err := usersClient.Get(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading user", common.FormatError(err))
		return
	}

	r.mapToState(ctx, user, &state, &resp.Diagnostics)
	state.ProjectID = types.StringValue(projectID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	usersClient := appwrite.NewUsers(r.clients.ClientForProject(projectID))

	var current userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &current)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := plan.ID.ValueString()
	var user *models.User

	if !plan.Name.IsNull() && plan.Name != current.Name {
		user, err = usersClient.UpdateName(id, plan.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error updating user name", common.FormatError(err))
			return
		}
	}
	if !plan.Email.IsNull() && plan.Email != current.Email {
		user, err = usersClient.UpdateEmail(id, plan.Email.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error updating user email", common.FormatError(err))
			return
		}
	}
	if !plan.Phone.IsNull() && plan.Phone != current.Phone {
		user, err = usersClient.UpdatePhone(id, plan.Phone.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error updating user phone", common.FormatError(err))
			return
		}
	}
	if !plan.Password.IsNull() && plan.Password != current.Password {
		user, err = usersClient.UpdatePassword(id, plan.Password.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error updating user password", common.FormatError(err))
			return
		}
	}
	if !plan.Status.IsNull() && plan.Status != current.Status {
		user, err = usersClient.UpdateStatus(id, plan.Status.ValueBool())
		if err != nil {
			resp.Diagnostics.AddError("Error updating user status", common.FormatError(err))
			return
		}
	}
	if !plan.Labels.IsNull() {
		var labels []string
		resp.Diagnostics.Append(plan.Labels.ElementsAs(ctx, &labels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		user, err = usersClient.UpdateLabels(id, labels)
		if err != nil {
			resp.Diagnostics.AddError("Error updating user labels", common.FormatError(err))
			return
		}
	}
	if !plan.EmailVerification.IsNull() && plan.EmailVerification != current.EmailVerification {
		user, err = usersClient.UpdateEmailVerification(id, plan.EmailVerification.ValueBool())
		if err != nil {
			resp.Diagnostics.AddError("Error updating email verification", common.FormatError(err))
			return
		}
	}
	if !plan.PhoneVerification.IsNull() && plan.PhoneVerification != current.PhoneVerification {
		user, err = usersClient.UpdatePhoneVerification(id, plan.PhoneVerification.ValueBool())
		if err != nil {
			resp.Diagnostics.AddError("Error updating phone verification", common.FormatError(err))
			return
		}
	}

	if user == nil {
		user, err = usersClient.Get(id)
		if err != nil {
			resp.Diagnostics.AddError("Error reading user", common.FormatError(err))
			return
		}
	}

	r.mapToState(ctx, user, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	usersClient := appwrite.NewUsers(r.clients.ClientForProject(projectID))

	_, err = usersClient.Delete(state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting user", common.FormatError(err))
	}
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *userResource) mapToState(ctx context.Context, user *models.User, model *userResourceModel, diagnostics *diag.Diagnostics) {
	model.ID = types.StringValue(user.Id)
	model.CreatedAt = types.StringValue(user.CreatedAt)
	model.UpdatedAt = types.StringValue(user.UpdatedAt)
	model.Status = types.BoolValue(user.Status)
	model.EmailVerification = types.BoolValue(user.EmailVerification)
	model.PhoneVerification = types.BoolValue(user.PhoneVerification)

	if user.Name != "" {
		model.Name = types.StringValue(user.Name)
	}
	if user.Email != "" {
		model.Email = types.StringValue(user.Email)
	}
	if user.Phone != "" {
		model.Phone = types.StringValue(user.Phone)
	}

	labelsList, diags := types.ListValueFrom(ctx, types.StringType, user.Labels)
	diagnostics.Append(diags...)
	model.Labels = labelsList
}
