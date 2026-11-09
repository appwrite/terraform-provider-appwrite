package project

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &smtpResource{}
	_ resource.ResourceWithConfigure   = &smtpResource{}
	_ resource.ResourceWithImportState = &smtpResource{}
	_ resource.ResourceWithModifyPlan  = &smtpResource{}
)

type smtpResource struct {
	clients *common.AppwriteClients
}

type smtpResourceModel struct {
	ID                types.String `tfsdk:"id"`
	ProjectID         types.String `tfsdk:"project_id"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	Host              types.String `tfsdk:"host"`
	Port              types.Int64  `tfsdk:"port"`
	Username          types.String `tfsdk:"username"`
	Password          types.String `tfsdk:"password"`
	PasswordWO        types.String `tfsdk:"password_wo"`
	PasswordWOVersion types.Int64  `tfsdk:"password_wo_version"`
	SenderEmail       types.String `tfsdk:"sender_email"`
	SenderName        types.String `tfsdk:"sender_name"`
	ReplyToEmail      types.String `tfsdk:"reply_to_email"`
	ReplyToName       types.String `tfsdk:"reply_to_name"`
	Secure            types.String `tfsdk:"secure"`
}

func NewSMTPResource() resource.Resource {
	return &smtpResource{}
}

func (r *smtpResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_smtp"
}

func (r *smtpResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a project's custom SMTP server, which Appwrite sends the project's emails through instead of its own.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the project the SMTP configuration belongs to.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": common.ProjectIDAttribute(),
			"enabled": schema.BoolAttribute{
				Description: "Whether Appwrite sends the project's emails through this server. While it is true, " +
					"Appwrite connects to the server and authenticates whenever the configuration is saved, and " +
					"rejects a configuration that fails. Defaults to true.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"host": schema.StringAttribute{
				Description: "The SMTP server's host name or IP address, without a scheme or port.",
				Required:    true,
				Validators:  smtpHostValidators(),
			},
			"port": schema.Int64Attribute{
				Description: "The SMTP server's port.",
				Required:    true,
				Validators:  []validator.Int64{int64validator.Between(1, 65535)},
			},
			"username": schema.StringAttribute{
				Description: "The username to authenticate with. Without one, Appwrite does not authenticate.",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.LengthAtMost(256)},
			},
			"password": schema.StringAttribute{
				Description: "The password to authenticate with. Stored in Terraform state; prefer password_wo, " +
					"which is not. Appwrite never returns the password, so a change made outside Terraform is not " +
					"detected. Leaving out both password and password_wo clears it.",
				Optional:  true,
				Sensitive: true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(256),
					stringvalidator.ConflictsWith(path.MatchRoot("password_wo")),
				},
			},
			"password_wo": schema.StringAttribute{
				Description: "The password to authenticate with, as a write-only argument. Read from the " +
					"configuration during apply and never persisted. Change password_wo_version to apply a new " +
					"value, since Terraform cannot detect a change in a value it does not store. Requires " +
					"Terraform 1.11 or later.",
				Optional:  true,
				Sensitive: true,
				WriteOnly: true,
				// The version is what makes a changed secret visible to
				// Terraform. Without it, switching from password to this
				// argument leaves both versions null, nothing compares unequal,
				// and the apply reports success while the old password stays.
				Validators: []validator.String{
					stringvalidator.LengthAtMost(256),
					stringvalidator.AlsoRequires(path.MatchRoot("password_wo_version")),
				},
			},
			"password_wo_version": schema.Int64Attribute{
				Description: "Increment to apply a changed password_wo.",
				Optional:    true,
			},
			"sender_email": schema.StringAttribute{
				Description: "The email address emails are sent from.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"sender_name": schema.StringAttribute{
				Description: "The name emails are sent from.",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.LengthAtMost(256)},
			},
			"reply_to_email": schema.StringAttribute{
				Description: "The email address replies go to.",
				Optional:    true,
			},
			"reply_to_name": schema.StringAttribute{
				Description: "The name replies go to.",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.LengthAtMost(256)},
			},
			"secure": schema.StringAttribute{
				Description: "The encryption to use: tls for STARTTLS, or ssl for implicit TLS. Without it, the " +
					"connection is not encrypted. Appwrite cannot remove this setting once it is stored, so it can " +
					"be changed between tls and ssl but not removed again.",
				Optional:   true,
				Validators: []validator.String{stringvalidator.OneOf(smtpSecureValues...)},
			},
		},
	}
}

func (r *smtpResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan refuses to plan removing secure from a configuration that set it.
// The API offers no way to clear it, so the apply would send nothing, the
// server would keep the old value, and Terraform would report the provider
// as inconsistent. Failing at plan time says why instead.
func (r *smtpResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var state, plan smtpResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := checkSecureRemoval(state.Secure.ValueString(), plan.Secure); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("secure"), "Cannot remove secure", err.Error())
	}
}

// checkSecureRemoval reports an error when the planned secure value would
// leave the server's stored one in place.
func checkSecureRemoval(stored string, planned types.String) error {
	if stored == "" || !planned.IsNull() {
		return nil
	}
	return fmt.Errorf("the project's SMTP configuration has secure = %q, and Appwrite has no way to remove it: "+
		"it accepts only tls or ssl. Set secure to tls or ssl", stored)
}

func (r *smtpResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan smtpResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	client, err := projectSettingsClient(r.clients, smtpResourceName, projectID, smtpScopes...)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	// The configuration already exists, since every project has one. A secure
	// value stored before Terraform took over cannot be removed, so a plan
	// without it can never be applied; say so before changing anything.
	current, err := client.Get()
	if err != nil {
		resp.Diagnostics.AddError("Error reading project SMTP configuration", r.formatError(err))
		return
	}
	if err := checkSecureRemoval(current.SmtpSecure, plan.Secure); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("secure"), "Cannot remove secure", err.Error())
		return
	}

	password := common.WriteOnlyValue(ctx, req.Config, "password_wo", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := updateSMTP(client, plan.settings(common.ResolveSecret(password, plan.Password)))
	if err != nil {
		resp.Diagnostics.AddError("Error configuring project SMTP", r.formatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	mapSMTPToState(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *smtpResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state smtpResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	client, err := projectSettingsClient(r.clients, smtpResourceName, projectID, smtpScopes...)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	current, err := client.Get()
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading project SMTP configuration", r.formatError(err))
		return
	}
	// A project with no host has never been configured, or was reset in a way
	// Terraform did not do. Either way there is nothing here to manage.
	if !smtpConfigured(current) {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ProjectID = types.StringValue(projectID)
	mapSMTPToState(current, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *smtpResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan smtpResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	client, err := projectSettingsClient(r.clients, smtpResourceName, projectID, smtpScopes...)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	password := common.WriteOnlyValue(ctx, req.Config, "password_wo", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := updateSMTP(client, plan.settings(common.ResolveSecret(password, plan.Password)))
	if err != nil {
		resp.Diagnostics.AddError("Error updating project SMTP", r.formatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	mapSMTPToState(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete disables custom SMTP, since the configuration itself cannot be
// deleted. See resetSMTP for what it clears.
func (r *smtpResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state smtpResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	client, err := projectSettingsClient(r.clients, smtpResourceName, projectID, smtpScopes...)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	if _, err := resetSMTP(client); err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error disabling project SMTP", r.formatError(err))
	}
}

// ImportState takes the project ID, which is also the resource's ID.
func (r *smtpResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Invalid import ID", "Expected the project ID")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), req.ID)...)
}

func (r *smtpResource) formatError(err error) string {
	return common.FormatErrorWithAuthGuidance(err, common.ProjectCredentialGuidance(smtpResourceName, smtpScopes...))
}

// settings builds the update request. Optional attributes left out of the
// configuration are sent empty, which is what clears them.
func (m smtpResourceModel) settings(password string) smtpSettings {
	return smtpSettings{
		Enabled:      m.Enabled.ValueBool(),
		Host:         m.Host.ValueString(),
		Port:         int(m.Port.ValueInt64()),
		Username:     m.Username.ValueString(),
		Password:     password,
		SenderEmail:  m.SenderEmail.ValueString(),
		SenderName:   m.SenderName.ValueString(),
		ReplyToEmail: m.ReplyToEmail.ValueString(),
		ReplyToName:  m.ReplyToName.ValueString(),
		Secure:       m.Secure.ValueString(),
	}
}

// mapSMTPToState copies the project's SMTP fields into the model. The password
// is left alone: Appwrite always returns it empty.
func mapSMTPToState(p *models.Project, model *smtpResourceModel) {
	model.ID = types.StringValue(p.Id)
	model.Enabled = types.BoolValue(p.SmtpEnabled)
	model.Host = types.StringValue(p.SmtpHost)
	model.Port = types.Int64Value(int64(p.SmtpPort))
	model.Username = optionalString(model.Username, p.SmtpUsername)
	model.SenderEmail = types.StringValue(p.SmtpSenderEmail)
	model.SenderName = optionalString(model.SenderName, p.SmtpSenderName)
	model.ReplyToEmail = optionalString(model.ReplyToEmail, p.SmtpReplyToEmail)
	model.ReplyToName = optionalString(model.ReplyToName, p.SmtpReplyToName)
	model.Secure = optionalString(model.Secure, p.SmtpSecure)
}
