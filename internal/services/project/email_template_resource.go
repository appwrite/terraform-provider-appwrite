package project

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v7/models"
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
	_ resource.Resource                = &emailTemplateResource{}
	_ resource.ResourceWithConfigure   = &emailTemplateResource{}
	_ resource.ResourceWithImportState = &emailTemplateResource{}
)

type emailTemplateResource struct {
	clients *common.AppwriteClients
}

type emailTemplateResourceModel struct {
	ID           types.String `tfsdk:"id"`
	ProjectID    types.String `tfsdk:"project_id"`
	Type         types.String `tfsdk:"type"`
	Locale       types.String `tfsdk:"locale"`
	Subject      types.String `tfsdk:"subject"`
	Message      types.String `tfsdk:"message"`
	SenderName   types.String `tfsdk:"sender_name"`
	SenderEmail  types.String `tfsdk:"sender_email"`
	ReplyToEmail types.String `tfsdk:"reply_to_email"`
	ReplyToName  types.String `tfsdk:"reply_to_name"`
}

func NewEmailTemplateResource() resource.Resource {
	return &emailTemplateResource{}
}

func (r *emailTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_email_template"
}

func (r *emailTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Manages a project's custom email template for one type of email in one locale. Appwrite uses " +
			"custom templates only while the project's custom SMTP is enabled.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The template's ID, in the form type/locale.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": common.ProjectIDAttribute(),
			"type": schema.StringAttribute{
				Description: "The email the template is for: verification, magicSession, recovery, invitation, " +
					"mfaChallenge, sessionAlert or otpSession. " + common.ForcesReplacementNote,
				Required:      true,
				PlanModifiers: replace,
				Validators:    []validator.String{stringvalidator.OneOf(emailTemplateTypes...)},
			},
			"locale": schema.StringAttribute{
				Description:   "The locale the template is for, such as en or pt-br. " + common.ForcesReplacementNote,
				Required:      true,
				PlanModifiers: replace,
				Validators:    []validator.String{stringvalidator.OneOf(emailTemplateLocales...)},
			},
			"subject": schema.StringAttribute{
				Description: "The email's subject.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 255)},
			},
			"message": schema.StringAttribute{
				Description: "The email's body, as plain text or HTML. Appwrite fills in placeholders such as " +
					"{{user}}, {{project}} and {{redirect}} when it sends the email.",
				Required:   true,
				Validators: []validator.String{stringvalidator.LengthBetween(1, 10485760)},
			},
			"sender_name": schema.StringAttribute{
				Description: "The name the email is sent from, overriding the SMTP configuration's sender_name.",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.LengthAtMost(255)},
			},
			"sender_email": schema.StringAttribute{
				Description: "The email address the email is sent from, overriding the SMTP configuration's sender_email.",
				Optional:    true,
			},
			"reply_to_email": schema.StringAttribute{
				Description: "The email address replies go to, overriding the SMTP configuration's reply_to_email.",
				Optional:    true,
			},
			"reply_to_name": schema.StringAttribute{
				Description: "The name replies go to, overriding the SMTP configuration's reply_to_name.",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.LengthAtMost(255)},
			},
		},
	}
}

func (r *emailTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create writes the custom template. There is nothing to create as such: every
// type and locale already has Appwrite's default, which this replaces.
func (r *emailTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan emailTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	client, err := projectSettingsClient(r.clients, templateResourceName, projectID, templateScopes...)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	updated, err := updateEmailTemplate(client, plan.settings())
	if err != nil {
		resp.Diagnostics.AddError("Error creating email template", r.formatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	mapEmailTemplateToState(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *emailTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan emailTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	client, err := projectSettingsClient(r.clients, templateResourceName, projectID, templateScopes...)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	updated, err := updateEmailTemplate(client, plan.settings())
	if err != nil {
		resp.Diagnostics.AddError("Error updating email template", r.formatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	mapEmailTemplateToState(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *emailTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state emailTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	client, err := projectSettingsClient(r.clients, templateResourceName, projectID, templateScopes...)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	found, err := findCustomEmailTemplate(client, state.Type.ValueString(), state.Locale.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading email template", r.formatError(err))
		return
	}
	// No custom template means the project is back on Appwrite's default,
	// most likely reset in the Console.
	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ProjectID = types.StringValue(projectID)
	mapEmailTemplateToState(found, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete leaves the template in place. Appwrite has no route a project API key
// can use to remove a custom template and fall back to the default: the only
// one, kept for backwards compatibility, is reserved for the Console. Saying
// so in a warning is better than an error, which would leave the resource
// impossible to destroy.
func (r *emailTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state emailTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.AddWarning(
		"Email template left in place",
		fmt.Sprintf("Appwrite has no API for a project key to remove a custom email template, so the %s template "+
			"for locale %s keeps its last applied content and Terraform no longer manages it. Reset it in the "+
			"Appwrite Console to restore the default. Custom templates are used only while custom SMTP is enabled.",
			state.Type.ValueString(), state.Locale.ValueString()),
	)
}

func (r *emailTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectID, templateType, locale, err := parseEmailTemplateImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	if projectID != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectID)...)
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), emailTemplateID(templateType, locale))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), templateType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("locale"), locale)...)
}

func (r *emailTemplateResource) formatError(err error) string {
	return common.FormatErrorWithAuthGuidance(err, common.ProjectCredentialGuidance(templateResourceName, templateScopes...))
}

func (m emailTemplateResourceModel) settings() emailTemplateSettings {
	return emailTemplateSettings{
		Type:         m.Type.ValueString(),
		Locale:       m.Locale.ValueString(),
		Subject:      m.Subject.ValueString(),
		Message:      m.Message.ValueString(),
		SenderName:   m.SenderName.ValueString(),
		SenderEmail:  m.SenderEmail.ValueString(),
		ReplyToEmail: m.ReplyToEmail.ValueString(),
		ReplyToName:  m.ReplyToName.ValueString(),
	}
}

func mapEmailTemplateToState(template *models.EmailTemplate, model *emailTemplateResourceModel) {
	model.ID = types.StringValue(emailTemplateID(template.TemplateId, template.Locale))
	model.Type = types.StringValue(template.TemplateId)
	model.Locale = types.StringValue(template.Locale)
	model.Subject = types.StringValue(template.Subject)
	model.Message = types.StringValue(template.Message)
	model.SenderName = optionalString(model.SenderName, template.SenderName)
	model.SenderEmail = optionalString(model.SenderEmail, template.SenderEmail)
	model.ReplyToEmail = optionalString(model.ReplyToEmail, template.ReplyToEmail)
	model.ReplyToName = optionalString(model.ReplyToName, template.ReplyToName)
}
