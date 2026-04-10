package provider

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v2/id"
	"github.com/appwrite/sdk-for-go/v2/messaging"
	"github.com/appwrite/sdk-for-go/v2/models"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &providerResource{}
	_ resource.ResourceWithConfigure   = &providerResource{}
	_ resource.ResourceWithImportState = &providerResource{}
)

type providerResource struct {
	messaging *messaging.Messaging
}

type providerResourceModel struct {
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
	ApiKey         types.String `tfsdk:"api_key"`
	ApiSecret      types.String `tfsdk:"api_secret"`
	Domain         types.String `tfsdk:"domain"`
	IsEuRegion     types.Bool   `tfsdk:"is_eu_region"`
	AccountSid     types.String `tfsdk:"account_sid"`
	AuthToken      types.String `tfsdk:"auth_token"`
	From           types.String `tfsdk:"from"`
	SenderId       types.String `tfsdk:"sender_id"`
	AuthKey        types.String `tfsdk:"auth_key"`
	AuthKeyId      types.String `tfsdk:"auth_key_id"`
	TeamId         types.String `tfsdk:"team_id"`
	BundleId       types.String `tfsdk:"bundle_id"`
	Sandbox        types.Bool   `tfsdk:"sandbox"`
	ServiceAccount types.String `tfsdk:"service_account_json"`
	CustomerId     types.String `tfsdk:"customer_id"`
	TemplateId     types.String `tfsdk:"template_id"`
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
			"id": schema.StringAttribute{
				Description:   "The provider ID.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The provider name.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description:   "The provider type. One of: sendgrid, mailgun, smtp, resend, twilio, vonage, msg91, telesign, textmagic, apns, fcm.",
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
			"created_at": schema.StringAttribute{
				Description: "The provider creation timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The provider last update timestamp in ISO 8601 format.",
				Computed:    true,
			},
		},
	}
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
	r.messaging = clients.Messaging
}

func (r *providerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan providerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	provID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		provID = id.Unique()
	}
	name := plan.Name.ValueString()
	providerType := plan.Type.ValueString()

	var prov *models.Provider
	var err error

	switch providerType {
	case "sendgrid":
		var opts []messaging.CreateSendgridProviderOption
		if v := plan.ApiKey; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateSendgridProviderApiKey(v.ValueString()))
		}
		if v := plan.FromEmail; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateSendgridProviderFromEmail(v.ValueString()))
		}
		if v := plan.FromName; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateSendgridProviderFromName(v.ValueString()))
		}
		if v := plan.ReplyToEmail; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateSendgridProviderReplyToEmail(v.ValueString()))
		}
		if v := plan.ReplyToName; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateSendgridProviderReplyToName(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, r.messaging.WithCreateSendgridProviderEnabled(v.ValueBool()))
		}
		prov, err = r.messaging.CreateSendgridProvider(provID, name, opts...)

	case "mailgun":
		var opts []messaging.CreateMailgunProviderOption
		if v := plan.ApiKey; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateMailgunProviderApiKey(v.ValueString()))
		}
		if v := plan.Domain; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateMailgunProviderDomain(v.ValueString()))
		}
		if v := plan.IsEuRegion; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateMailgunProviderIsEuRegion(v.ValueBool()))
		}
		if v := plan.FromEmail; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateMailgunProviderFromEmail(v.ValueString()))
		}
		if v := plan.FromName; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateMailgunProviderFromName(v.ValueString()))
		}
		if v := plan.ReplyToEmail; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateMailgunProviderReplyToEmail(v.ValueString()))
		}
		if v := plan.ReplyToName; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateMailgunProviderReplyToName(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, r.messaging.WithCreateMailgunProviderEnabled(v.ValueBool()))
		}
		prov, err = r.messaging.CreateMailgunProvider(provID, name, opts...)

	case "smtp":
		if plan.Host.IsNull() {
			resp.Diagnostics.AddError("Missing attribute", "host is required for smtp providers")
			return
		}
		var opts []messaging.CreateSmtpProviderOption
		if v := plan.Port; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateSmtpProviderPort(int(v.ValueInt64())))
		}
		if v := plan.Username; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateSmtpProviderUsername(v.ValueString()))
		}
		if v := plan.Password; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateSmtpProviderPassword(v.ValueString()))
		}
		if v := plan.Encryption; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateSmtpProviderEncryption(v.ValueString()))
		}
		if v := plan.AutoTLS; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateSmtpProviderAutoTLS(v.ValueBool()))
		}
		if v := plan.FromEmail; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateSmtpProviderFromEmail(v.ValueString()))
		}
		if v := plan.FromName; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateSmtpProviderFromName(v.ValueString()))
		}
		if v := plan.ReplyToEmail; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateSmtpProviderReplyToEmail(v.ValueString()))
		}
		if v := plan.ReplyToName; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateSmtpProviderReplyToName(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, r.messaging.WithCreateSmtpProviderEnabled(v.ValueBool()))
		}
		prov, err = r.messaging.CreateSmtpProvider(provID, name, plan.Host.ValueString(), opts...)

	case "resend":
		var opts []messaging.CreateResendProviderOption
		if v := plan.ApiKey; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateResendProviderApiKey(v.ValueString()))
		}
		if v := plan.FromEmail; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateResendProviderFromEmail(v.ValueString()))
		}
		if v := plan.FromName; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateResendProviderFromName(v.ValueString()))
		}
		if v := plan.ReplyToEmail; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateResendProviderReplyToEmail(v.ValueString()))
		}
		if v := plan.ReplyToName; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateResendProviderReplyToName(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, r.messaging.WithCreateResendProviderEnabled(v.ValueBool()))
		}
		prov, err = r.messaging.CreateResendProvider(provID, name, opts...)

	case "twilio":
		var opts []messaging.CreateTwilioProviderOption
		if v := plan.AccountSid; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateTwilioProviderAccountSid(v.ValueString()))
		}
		if v := plan.AuthToken; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateTwilioProviderAuthToken(v.ValueString()))
		}
		if v := plan.From; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateTwilioProviderFrom(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, r.messaging.WithCreateTwilioProviderEnabled(v.ValueBool()))
		}
		prov, err = r.messaging.CreateTwilioProvider(provID, name, opts...)

	case "vonage":
		var opts []messaging.CreateVonageProviderOption
		if v := plan.ApiKey; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateVonageProviderApiKey(v.ValueString()))
		}
		if v := plan.ApiSecret; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateVonageProviderApiSecret(v.ValueString()))
		}
		if v := plan.From; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateVonageProviderFrom(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, r.messaging.WithCreateVonageProviderEnabled(v.ValueBool()))
		}
		prov, err = r.messaging.CreateVonageProvider(provID, name, opts...)

	case "msg91":
		var opts []messaging.CreateMsg91ProviderOption
		if v := plan.SenderId; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateMsg91ProviderSenderId(v.ValueString()))
		}
		if v := plan.AuthKey; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateMsg91ProviderAuthKey(v.ValueString()))
		}
		if v := plan.TemplateId; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateMsg91ProviderTemplateId(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, r.messaging.WithCreateMsg91ProviderEnabled(v.ValueBool()))
		}
		prov, err = r.messaging.CreateMsg91Provider(provID, name, opts...)

	case "telesign":
		var opts []messaging.CreateTelesignProviderOption
		if v := plan.CustomerId; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateTelesignProviderCustomerId(v.ValueString()))
		}
		if v := plan.ApiKey; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateTelesignProviderApiKey(v.ValueString()))
		}
		if v := plan.From; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateTelesignProviderFrom(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, r.messaging.WithCreateTelesignProviderEnabled(v.ValueBool()))
		}
		prov, err = r.messaging.CreateTelesignProvider(provID, name, opts...)

	case "textmagic":
		var opts []messaging.CreateTextmagicProviderOption
		if v := plan.Username; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateTextmagicProviderUsername(v.ValueString()))
		}
		if v := plan.ApiKey; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateTextmagicProviderApiKey(v.ValueString()))
		}
		if v := plan.From; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateTextmagicProviderFrom(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, r.messaging.WithCreateTextmagicProviderEnabled(v.ValueBool()))
		}
		prov, err = r.messaging.CreateTextmagicProvider(provID, name, opts...)

	case "apns":
		var opts []messaging.CreateApnsProviderOption
		if v := plan.AuthKey; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateApnsProviderAuthKey(v.ValueString()))
		}
		if v := plan.AuthKeyId; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateApnsProviderAuthKeyId(v.ValueString()))
		}
		if v := plan.TeamId; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateApnsProviderTeamId(v.ValueString()))
		}
		if v := plan.BundleId; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateApnsProviderBundleId(v.ValueString()))
		}
		if v := plan.Sandbox; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateApnsProviderSandbox(v.ValueBool()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, r.messaging.WithCreateApnsProviderEnabled(v.ValueBool()))
		}
		prov, err = r.messaging.CreateApnsProvider(provID, name, opts...)

	case "fcm":
		var opts []messaging.CreateFcmProviderOption
		if v := plan.ServiceAccount; !v.IsNull() {
			opts = append(opts, r.messaging.WithCreateFcmProviderServiceAccountJSON(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, r.messaging.WithCreateFcmProviderEnabled(v.ValueBool()))
		}
		prov, err = r.messaging.CreateFcmProvider(provID, name, opts...)

	default:
		resp.Diagnostics.AddError("Unsupported provider type", fmt.Sprintf("Provider type %q is not supported.", providerType))
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Error creating messaging provider", common.FormatError(err))
		return
	}

	r.mapToState(prov, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *providerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state providerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	prov, err := r.messaging.GetProvider(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading messaging provider", common.FormatError(err))
		return
	}

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

	id := plan.ID.ValueString()
	providerType := plan.Type.ValueString()

	var prov *models.Provider
	var err error

	switch providerType {
	case "sendgrid":
		var opts []messaging.UpdateSendgridProviderOption
		if v := plan.Name; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateSendgridProviderName(v.ValueString()))
		}
		if v := plan.ApiKey; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateSendgridProviderApiKey(v.ValueString()))
		}
		if v := plan.FromEmail; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateSendgridProviderFromEmail(v.ValueString()))
		}
		if v := plan.FromName; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateSendgridProviderFromName(v.ValueString()))
		}
		if v := plan.ReplyToEmail; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateSendgridProviderReplyToEmail(v.ValueString()))
		}
		if v := plan.ReplyToName; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateSendgridProviderReplyToName(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, r.messaging.WithUpdateSendgridProviderEnabled(v.ValueBool()))
		}
		prov, err = r.messaging.UpdateSendgridProvider(id, opts...)

	case "mailgun":
		var opts []messaging.UpdateMailgunProviderOption
		if v := plan.Name; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateMailgunProviderName(v.ValueString()))
		}
		if v := plan.ApiKey; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateMailgunProviderApiKey(v.ValueString()))
		}
		if v := plan.Domain; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateMailgunProviderDomain(v.ValueString()))
		}
		if v := plan.IsEuRegion; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateMailgunProviderIsEuRegion(v.ValueBool()))
		}
		if v := plan.FromEmail; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateMailgunProviderFromEmail(v.ValueString()))
		}
		if v := plan.FromName; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateMailgunProviderFromName(v.ValueString()))
		}
		if v := plan.ReplyToEmail; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateMailgunProviderReplyToEmail(v.ValueString()))
		}
		if v := plan.ReplyToName; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateMailgunProviderReplyToName(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, r.messaging.WithUpdateMailgunProviderEnabled(v.ValueBool()))
		}
		prov, err = r.messaging.UpdateMailgunProvider(id, opts...)

	case "smtp":
		var opts []messaging.UpdateSmtpProviderOption
		if v := plan.Name; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateSmtpProviderName(v.ValueString()))
		}
		if v := plan.Host; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateSmtpProviderHost(v.ValueString()))
		}
		if v := plan.Port; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateSmtpProviderPort(int(v.ValueInt64())))
		}
		if v := plan.Username; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateSmtpProviderUsername(v.ValueString()))
		}
		if v := plan.Password; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateSmtpProviderPassword(v.ValueString()))
		}
		if v := plan.Encryption; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateSmtpProviderEncryption(v.ValueString()))
		}
		if v := plan.AutoTLS; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateSmtpProviderAutoTLS(v.ValueBool()))
		}
		if v := plan.FromEmail; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateSmtpProviderFromEmail(v.ValueString()))
		}
		if v := plan.FromName; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateSmtpProviderFromName(v.ValueString()))
		}
		if v := plan.ReplyToEmail; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateSmtpProviderReplyToEmail(v.ValueString()))
		}
		if v := plan.ReplyToName; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateSmtpProviderReplyToName(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, r.messaging.WithUpdateSmtpProviderEnabled(v.ValueBool()))
		}
		prov, err = r.messaging.UpdateSmtpProvider(id, opts...)

	case "resend":
		var opts []messaging.UpdateResendProviderOption
		if v := plan.Name; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateResendProviderName(v.ValueString()))
		}
		if v := plan.ApiKey; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateResendProviderApiKey(v.ValueString()))
		}
		if v := plan.FromEmail; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateResendProviderFromEmail(v.ValueString()))
		}
		if v := plan.FromName; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateResendProviderFromName(v.ValueString()))
		}
		if v := plan.ReplyToEmail; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateResendProviderReplyToEmail(v.ValueString()))
		}
		if v := plan.ReplyToName; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateResendProviderReplyToName(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, r.messaging.WithUpdateResendProviderEnabled(v.ValueBool()))
		}
		prov, err = r.messaging.UpdateResendProvider(id, opts...)

	case "twilio":
		var opts []messaging.UpdateTwilioProviderOption
		if v := plan.Name; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateTwilioProviderName(v.ValueString()))
		}
		if v := plan.AccountSid; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateTwilioProviderAccountSid(v.ValueString()))
		}
		if v := plan.AuthToken; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateTwilioProviderAuthToken(v.ValueString()))
		}
		if v := plan.From; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateTwilioProviderFrom(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, r.messaging.WithUpdateTwilioProviderEnabled(v.ValueBool()))
		}
		prov, err = r.messaging.UpdateTwilioProvider(id, opts...)

	case "vonage":
		var opts []messaging.UpdateVonageProviderOption
		if v := plan.Name; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateVonageProviderName(v.ValueString()))
		}
		if v := plan.ApiKey; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateVonageProviderApiKey(v.ValueString()))
		}
		if v := plan.ApiSecret; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateVonageProviderApiSecret(v.ValueString()))
		}
		if v := plan.From; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateVonageProviderFrom(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, r.messaging.WithUpdateVonageProviderEnabled(v.ValueBool()))
		}
		prov, err = r.messaging.UpdateVonageProvider(id, opts...)

	case "msg91":
		var opts []messaging.UpdateMsg91ProviderOption
		if v := plan.Name; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateMsg91ProviderName(v.ValueString()))
		}
		if v := plan.SenderId; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateMsg91ProviderSenderId(v.ValueString()))
		}
		if v := plan.AuthKey; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateMsg91ProviderAuthKey(v.ValueString()))
		}
		if v := plan.TemplateId; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateMsg91ProviderTemplateId(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, r.messaging.WithUpdateMsg91ProviderEnabled(v.ValueBool()))
		}
		prov, err = r.messaging.UpdateMsg91Provider(id, opts...)

	case "telesign":
		var opts []messaging.UpdateTelesignProviderOption
		if v := plan.Name; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateTelesignProviderName(v.ValueString()))
		}
		if v := plan.CustomerId; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateTelesignProviderCustomerId(v.ValueString()))
		}
		if v := plan.ApiKey; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateTelesignProviderApiKey(v.ValueString()))
		}
		if v := plan.From; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateTelesignProviderFrom(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, r.messaging.WithUpdateTelesignProviderEnabled(v.ValueBool()))
		}
		prov, err = r.messaging.UpdateTelesignProvider(id, opts...)

	case "textmagic":
		var opts []messaging.UpdateTextmagicProviderOption
		if v := plan.Name; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateTextmagicProviderName(v.ValueString()))
		}
		if v := plan.Username; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateTextmagicProviderUsername(v.ValueString()))
		}
		if v := plan.ApiKey; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateTextmagicProviderApiKey(v.ValueString()))
		}
		if v := plan.From; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateTextmagicProviderFrom(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, r.messaging.WithUpdateTextmagicProviderEnabled(v.ValueBool()))
		}
		prov, err = r.messaging.UpdateTextmagicProvider(id, opts...)

	case "apns":
		var opts []messaging.UpdateApnsProviderOption
		if v := plan.Name; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateApnsProviderName(v.ValueString()))
		}
		if v := plan.AuthKey; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateApnsProviderAuthKey(v.ValueString()))
		}
		if v := plan.AuthKeyId; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateApnsProviderAuthKeyId(v.ValueString()))
		}
		if v := plan.TeamId; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateApnsProviderTeamId(v.ValueString()))
		}
		if v := plan.BundleId; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateApnsProviderBundleId(v.ValueString()))
		}
		if v := plan.Sandbox; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateApnsProviderSandbox(v.ValueBool()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, r.messaging.WithUpdateApnsProviderEnabled(v.ValueBool()))
		}
		prov, err = r.messaging.UpdateApnsProvider(id, opts...)

	case "fcm":
		var opts []messaging.UpdateFcmProviderOption
		if v := plan.Name; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateFcmProviderName(v.ValueString()))
		}
		if v := plan.ServiceAccount; !v.IsNull() {
			opts = append(opts, r.messaging.WithUpdateFcmProviderServiceAccountJSON(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, r.messaging.WithUpdateFcmProviderEnabled(v.ValueBool()))
		}
		prov, err = r.messaging.UpdateFcmProvider(id, opts...)
	}

	if err != nil {
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

	_, err := r.messaging.DeleteProvider(state.ID.ValueString())
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
	// Don't overwrite type from API — preserve the user's value
}
