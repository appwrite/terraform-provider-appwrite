package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/id"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                   = &providerResource{}
	_ resource.ResourceWithConfigure      = &providerResource{}
	_ resource.ResourceWithImportState    = &providerResource{}
	_ resource.ResourceWithValidateConfig = &providerResource{}
)

// maxPushExpiry is the longest an appwrite provider may keep a message for
// offline delivery: Appwrite's push ledger retention, 7 days.
const maxPushExpiry = 7 * 24 * 60 * 60

type providerResource struct {
	clients *common.AppwriteClients
}

type providerResourceModel struct {
	ProjectID      types.String `tfsdk:"project_id"`
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	FromEmail      types.String `tfsdk:"from_email"`
	FromName       types.String `tfsdk:"from_name"`
	ReplyToEmail   types.String `tfsdk:"reply_to_email"`
	ReplyToName    types.String `tfsdk:"reply_to_name"`
	Host           types.String `tfsdk:"host"`
	Port           types.Int64  `tfsdk:"port"`
	Username       types.String `tfsdk:"username"`
	Password       types.String `tfsdk:"password"`
	Encryption     types.String `tfsdk:"encryption"`
	AutoTLS        types.Bool   `tfsdk:"auto_tls"`
	APIKey         types.String `tfsdk:"api_key"`
	APISecret      types.String `tfsdk:"api_secret"`
	Domain         types.String `tfsdk:"domain"`
	IsEuRegion     types.Bool   `tfsdk:"is_eu_region"`
	AccountSid     types.String `tfsdk:"account_sid"`
	AuthToken      types.String `tfsdk:"auth_token"`
	From           types.String `tfsdk:"from"`
	SenderID       types.String `tfsdk:"sender_id"`
	AuthKey        types.String `tfsdk:"auth_key"`
	AuthKeyID      types.String `tfsdk:"auth_key_id"`
	TeamID         types.String `tfsdk:"team_id"`
	BundleID       types.String `tfsdk:"bundle_id"`
	Sandbox        types.Bool   `tfsdk:"sandbox"`
	ServiceAccount types.String `tfsdk:"service_account_json"`
	CustomerID     types.String `tfsdk:"customer_id"`
	TemplateID     types.String `tfsdk:"template_id"`
	AccessKey      types.String `tfsdk:"access_key"`
	SecretKey      types.String `tfsdk:"secret_key"`
	Region         types.String `tfsdk:"region"`
	Qos            types.Int64  `tfsdk:"qos"`
	Expiry         types.Int64  `tfsdk:"expiry"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func NewProviderResource() resource.Resource {
	return &providerResource{}
}

func (r *providerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_messaging_provider"
}

func (r *providerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Appwrite messaging provider.",
		Attributes: map[string]schema.Attribute{
			"project_id": common.ProjectIDAttribute(),
			"id": schema.StringAttribute{
				Description:   "The provider ID. Changing this forces a new resource to be created.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The provider name.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description:   "The provider type. One of: sendgrid, mailgun, smtp, resend, ses, twilio, vonage, msg91, telesign, textmagic, apns, fcm, appwrite. The appwrite type needs Appwrite 2.3.0 or later. Changing this forces a new resource to be created.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the provider is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			// Email provider fields
			"from_email": schema.StringAttribute{
				Description: "Sender email address. Applies to email providers.",
				Optional:    true,
			},
			"from_name": schema.StringAttribute{
				Description: "Sender name. Applies to email providers.",
				Optional:    true,
			},
			"reply_to_email": schema.StringAttribute{
				Description: "Reply-to email address. Applies to email providers.",
				Optional:    true,
			},
			"reply_to_name": schema.StringAttribute{
				Description: "Reply-to name. Applies to email providers.",
				Optional:    true,
			},
			// SMTP fields
			"host": schema.StringAttribute{
				Description: "SMTP host. Required for smtp type.",
				Optional:    true,
			},
			"port": schema.Int64Attribute{
				Description: "SMTP port. Applies to smtp type.",
				Optional:    true,
			},
			"username": schema.StringAttribute{
				Description: "Username. Applies to smtp and textmagic types.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "Password. Applies to smtp type.",
				Optional:    true,
				Sensitive:   true,
			},
			"encryption": schema.StringAttribute{
				Description: "SMTP encryption: none, ssl, tls. Applies to smtp type.",
				Optional:    true,
			},
			"auto_tls": schema.BoolAttribute{
				Description: "SMTP auto TLS. Applies to smtp type.",
				Optional:    true,
			},
			// API key based providers
			"api_key": schema.StringAttribute{
				Description: "API key. Applies to sendgrid, mailgun, resend, vonage, telesign, textmagic types.",
				Optional:    true,
				Sensitive:   true,
			},
			"api_secret": schema.StringAttribute{
				Description: "API secret. Applies to vonage type.",
				Optional:    true,
				Sensitive:   true,
			},
			// Mailgun fields
			"domain": schema.StringAttribute{
				Description: "Mailgun domain. Applies to mailgun type.",
				Optional:    true,
			},
			"is_eu_region": schema.BoolAttribute{
				Description: "Whether to use EU region. Applies to mailgun type.",
				Optional:    true,
			},
			// Twilio fields
			"account_sid": schema.StringAttribute{
				Description: "Twilio account SID. Applies to twilio type.",
				Optional:    true,
			},
			"auth_token": schema.StringAttribute{
				Description: "Twilio auth token. Applies to twilio type.",
				Optional:    true,
				Sensitive:   true,
			},
			// SMS provider fields
			"from": schema.StringAttribute{
				Description: "Sender phone number or ID. Applies to SMS providers.",
				Optional:    true,
			},
			// Msg91 fields
			"sender_id": schema.StringAttribute{
				Description: "Msg91 sender ID. Applies to msg91 type.",
				Optional:    true,
			},
			"template_id": schema.StringAttribute{
				Description: "Msg91 template ID. Applies to msg91 type.",
				Optional:    true,
			},
			"customer_id": schema.StringAttribute{
				Description: "Telesign customer ID. Applies to telesign type.",
				Optional:    true,
			},
			// APNS fields
			"auth_key": schema.StringAttribute{
				Description: "APNS auth key content. Applies to apns type.",
				Optional:    true,
				Sensitive:   true,
			},
			"auth_key_id": schema.StringAttribute{
				Description: "APNS auth key ID. Applies to apns type.",
				Optional:    true,
			},
			"team_id": schema.StringAttribute{
				Description: "Apple team ID. Applies to apns type.",
				Optional:    true,
			},
			"bundle_id": schema.StringAttribute{
				Description: "iOS bundle ID. Applies to apns type.",
				Optional:    true,
			},
			"sandbox": schema.BoolAttribute{
				Description: "Use APNS sandbox environment. Applies to apns type.",
				Optional:    true,
			},
			// FCM fields
			"service_account_json": schema.StringAttribute{
				Description: "Firebase service account JSON. Applies to fcm type.",
				Optional:    true,
				Sensitive:   true,
			},
			// SES fields
			"access_key": schema.StringAttribute{
				Description: "AWS access key ID. Applies to ses type.",
				Optional:    true,
				Sensitive:   true,
			},
			"secret_key": schema.StringAttribute{
				Description: "AWS secret access key. Applies to ses type.",
				Optional:    true,
				Sensitive:   true,
			},
			"region": schema.StringAttribute{
				Description: "AWS region, for example us-east-1. Applies to ses type.",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 128)},
			},
			// Appwrite push fields
			"qos": schema.Int64Attribute{
				Description: "Default MQTT QoS, 0 or 1, for topics on this provider. Unset lets each subscriber choose. Applies to appwrite type.",
				Optional:    true,
				Validators:  []validator.Int64{int64validator.Between(0, 1)},
			},
			"expiry": schema.Int64Attribute{
				Description: fmt.Sprintf("Default number of seconds a message is kept for offline delivery, at most %d (7 days). Applies to topics with no expiry of their own; the lowest value across a project's enabled appwrite providers wins. Applies to appwrite type.", maxPushExpiry),
				Optional:    true,
				Validators:  []validator.Int64{int64validator.Between(0, maxPushExpiry)},
			},
			"created_at": schema.StringAttribute{
				Description: "The provider creation timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The provider last update timestamp in ISO 8601 format.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					common.UseStateForUnknownUnlessUpdating(),
				},
			},
		},
	}
}

// ValidateConfig rejects the ses and appwrite arguments on any other type.
//
// Only arguments added alongside those two types are checked. The older
// per-type arguments have always been accepted, and ignored, on a type they
// do not apply to, and rejecting them now would fail configurations that plan
// cleanly today.
func (r *providerResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config providerResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.Type.IsUnknown() || config.Type.IsNull() {
		return
	}
	resp.Diagnostics.Append(validateProviderConfig(config)...)
}

func validateProviderConfig(config providerResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	providerType := config.Type.ValueString()

	for _, field := range []struct {
		name      string
		value     attr.Value
		appliesTo string
	}{
		{"access_key", config.AccessKey, typeSES},
		{"secret_key", config.SecretKey, typeSES},
		{"region", config.Region, typeSES},
		{"qos", config.Qos, typeAppwrite},
		{"expiry", config.Expiry, typeAppwrite},
	} {
		// A value from another resource may not be known until apply, and
		// may turn out null. Only a value known now can be judged.
		if field.value.IsNull() || field.value.IsUnknown() || providerType == field.appliesTo {
			continue
		}
		diags.AddAttributeError(path.Root(field.name), "Unsupported argument",
			fmt.Sprintf("%s applies only to %s providers, not %s. Remove it from the configuration.", field.name, field.appliesTo, providerType))
	}
	return diags
}

func (r *providerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *providerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan providerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	messagingClient := appwrite.NewMessaging(r.clients.ClientForProject(projectID))

	provID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		provID = id.Unique()
	}

	prov, err := createProvider(messagingClient, provID, plan)
	switch {
	case errors.Is(err, errSMTPHostRequired):
		resp.Diagnostics.AddError("Missing attribute", err.Error())
		return
	case errors.Is(err, errUnsupportedType):
		resp.Diagnostics.AddError("Unsupported provider type", fmt.Sprintf("Provider type %q is not supported.", plan.Type.ValueString()))
		return
	case plan.Type.ValueString() == typeAppwrite && isRouteMissing(err):
		resp.Diagnostics.AddError("Error creating messaging provider",
			"This Appwrite server has no route for the appwrite provider type, which Appwrite 2.3.0 added. Upgrade the server to use it.")
		return
	case err != nil:
		resp.Diagnostics.AddError("Error creating messaging provider", common.FormatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(prov, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *providerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state providerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	messagingClient := appwrite.NewMessaging(r.clients.ClientForProject(projectID))

	prov, err := messagingClient.GetProvider(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading messaging provider", common.FormatError(err))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	r.mapToState(prov, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *providerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Provider updates are type-specific. For simplicity, we read back after update.
	// The API returns the updated provider in each case.
	var plan providerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	c := r.clients.ClientForProject(projectID)

	prov, err := updateProvider(c, appwrite.NewMessaging(c), plan)
	switch {
	case errors.Is(err, errUnsupportedType):
		resp.Diagnostics.AddError("Unsupported provider type", fmt.Sprintf("Provider type %q is not supported.", plan.Type.ValueString()))
		return
	case err != nil:
		resp.Diagnostics.AddError("Error updating messaging provider", common.FormatError(err))
		return
	}

	r.mapToState(prov, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *providerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state providerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	messagingClient := appwrite.NewMessaging(r.clients.ClientForProject(projectID))

	_, err = messagingClient.DeleteProvider(state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting messaging provider", common.FormatError(err))
	}
}

func (r *providerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *providerResource) mapToState(prov *models.Provider, model *providerResourceModel) {
	model.ID = types.StringValue(prov.Id)
	model.Name = types.StringValue(prov.Name)
	model.Enabled = types.BoolValue(prov.Enabled)
	model.CreatedAt = types.StringValue(prov.CreatedAt)
	model.UpdatedAt = types.StringValue(prov.UpdatedAt)
	// Don't overwrite type when already set — preserve the user's value.
	// During import, type is not yet in state, so populate it from the API.
	//
	// This schema's `type` is the provider service (sendgrid, smtp, twilio),
	// which the API returns as `provider`. The API's own `type` field is the
	// message category (email, sms, push) and is a different thing entirely.
	if model.Type.IsNull() || model.Type.IsUnknown() {
		model.Type = types.StringValue(prov.Provider)
	}

	// from_email and from_name are optional rather than computed, so they are
	// only filled in when state has nothing -- that is, on import. Setting them
	// on every read would manufacture a value the config never asked for.
	options, _ := prov.Options.(map[string]interface{})
	if s, ok := options["fromEmail"].(string); ok && s != "" && model.FromEmail.IsNull() {
		model.FromEmail = types.StringValue(s)
	}
	if s, ok := options["fromName"].(string); ok && s != "" && model.FromName.IsNull() {
		model.FromName = types.StringValue(s)
	}

	// qos and expiry are read back on every refresh, unlike the arguments
	// above. Appwrite stores null for one left unset, and an update that
	// leaves one unset sends null to clear it, so the server's value always
	// matches what the configuration asked for and a change made outside
	// Terraform shows up as drift.
	if prov.Provider == typeAppwrite {
		model.Qos = optionalInt64(options["qos"])
		model.Expiry = optionalInt64(options["expiry"])
	}
}

// optionalInt64 reads a nullable number from a provider's options. JSON
// numbers decode as float64.
func optionalInt64(v interface{}) types.Int64 {
	if f, ok := v.(float64); ok {
		return types.Int64Value(int64(f))
	}
	return types.Int64Null()
}
