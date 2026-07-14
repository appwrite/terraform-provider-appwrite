package provider

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v6/appwrite"
	"github.com/appwrite/sdk-for-go/v6/id"
	"github.com/appwrite/sdk-for-go/v6/messaging"
	"github.com/appwrite/sdk-for-go/v6/models"
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
				PlanModifiers: []planmodifier.String{
					common.UseStateForUnknownUnlessUpdating(),
				},
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
	name := plan.Name.ValueString()
	providerType := plan.Type.ValueString()

	var prov *models.Provider

	switch providerType {
	case "sendgrid":
		var opts []messaging.CreateSendgridProviderOption
		if v := plan.APIKey; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateSendgridProviderApiKey(v.ValueString()))
		}
		if v := plan.FromEmail; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateSendgridProviderFromEmail(v.ValueString()))
		}
		if v := plan.FromName; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateSendgridProviderFromName(v.ValueString()))
		}
		if v := plan.ReplyToEmail; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateSendgridProviderReplyToEmail(v.ValueString()))
		}
		if v := plan.ReplyToName; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateSendgridProviderReplyToName(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithCreateSendgridProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.CreateSendgridProvider(provID, name, opts...)

	case "mailgun":
		var opts []messaging.CreateMailgunProviderOption
		if v := plan.APIKey; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateMailgunProviderApiKey(v.ValueString()))
		}
		if v := plan.Domain; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateMailgunProviderDomain(v.ValueString()))
		}
		if v := plan.IsEuRegion; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateMailgunProviderIsEuRegion(v.ValueBool()))
		}
		if v := plan.FromEmail; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateMailgunProviderFromEmail(v.ValueString()))
		}
		if v := plan.FromName; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateMailgunProviderFromName(v.ValueString()))
		}
		if v := plan.ReplyToEmail; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateMailgunProviderReplyToEmail(v.ValueString()))
		}
		if v := plan.ReplyToName; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateMailgunProviderReplyToName(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithCreateMailgunProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.CreateMailgunProvider(provID, name, opts...)

	case "smtp":
		if plan.Host.IsNull() {
			resp.Diagnostics.AddError("Missing attribute", "host is required for smtp providers")
			return
		}
		var opts []messaging.CreateSMTPProviderOption
		if v := plan.Port; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateSMTPProviderPort(int(v.ValueInt64())))
		}
		if v := plan.Username; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateSMTPProviderUsername(v.ValueString()))
		}
		if v := plan.Password; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateSMTPProviderPassword(v.ValueString()))
		}
		if v := plan.Encryption; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateSMTPProviderEncryption(v.ValueString()))
		}
		if v := plan.AutoTLS; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateSMTPProviderAutoTLS(v.ValueBool()))
		}
		if v := plan.FromEmail; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateSMTPProviderFromEmail(v.ValueString()))
		}
		if v := plan.FromName; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateSMTPProviderFromName(v.ValueString()))
		}
		if v := plan.ReplyToEmail; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateSMTPProviderReplyToEmail(v.ValueString()))
		}
		if v := plan.ReplyToName; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateSMTPProviderReplyToName(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithCreateSMTPProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.CreateSMTPProvider(provID, name, plan.Host.ValueString(), opts...)

	case "resend":
		var opts []messaging.CreateResendProviderOption
		if v := plan.APIKey; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateResendProviderApiKey(v.ValueString()))
		}
		if v := plan.FromEmail; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateResendProviderFromEmail(v.ValueString()))
		}
		if v := plan.FromName; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateResendProviderFromName(v.ValueString()))
		}
		if v := plan.ReplyToEmail; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateResendProviderReplyToEmail(v.ValueString()))
		}
		if v := plan.ReplyToName; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateResendProviderReplyToName(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithCreateResendProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.CreateResendProvider(provID, name, opts...)

	case "twilio":
		var opts []messaging.CreateTwilioProviderOption
		if v := plan.AccountSid; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateTwilioProviderAccountSid(v.ValueString()))
		}
		if v := plan.AuthToken; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateTwilioProviderAuthToken(v.ValueString()))
		}
		if v := plan.From; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateTwilioProviderFrom(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithCreateTwilioProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.CreateTwilioProvider(provID, name, opts...)

	case "vonage":
		var opts []messaging.CreateVonageProviderOption
		if v := plan.APIKey; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateVonageProviderApiKey(v.ValueString()))
		}
		if v := plan.APISecret; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateVonageProviderApiSecret(v.ValueString()))
		}
		if v := plan.From; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateVonageProviderFrom(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithCreateVonageProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.CreateVonageProvider(provID, name, opts...)

	case "msg91":
		var opts []messaging.CreateMsg91ProviderOption
		if v := plan.SenderID; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateMsg91ProviderSenderId(v.ValueString()))
		}
		if v := plan.AuthKey; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateMsg91ProviderAuthKey(v.ValueString()))
		}
		if v := plan.TemplateID; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateMsg91ProviderTemplateId(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithCreateMsg91ProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.CreateMsg91Provider(provID, name, opts...)

	case "telesign":
		var opts []messaging.CreateTelesignProviderOption
		if v := plan.CustomerID; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateTelesignProviderCustomerId(v.ValueString()))
		}
		if v := plan.APIKey; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateTelesignProviderApiKey(v.ValueString()))
		}
		if v := plan.From; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateTelesignProviderFrom(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithCreateTelesignProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.CreateTelesignProvider(provID, name, opts...)

	case "textmagic":
		var opts []messaging.CreateTextmagicProviderOption
		if v := plan.Username; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateTextmagicProviderUsername(v.ValueString()))
		}
		if v := plan.APIKey; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateTextmagicProviderApiKey(v.ValueString()))
		}
		if v := plan.From; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateTextmagicProviderFrom(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithCreateTextmagicProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.CreateTextmagicProvider(provID, name, opts...)

	case "apns":
		var opts []messaging.CreateAPNSProviderOption
		if v := plan.AuthKey; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateAPNSProviderAuthKey(v.ValueString()))
		}
		if v := plan.AuthKeyID; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateAPNSProviderAuthKeyId(v.ValueString()))
		}
		if v := plan.TeamID; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateAPNSProviderTeamId(v.ValueString()))
		}
		if v := plan.BundleID; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateAPNSProviderBundleId(v.ValueString()))
		}
		if v := plan.Sandbox; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateAPNSProviderSandbox(v.ValueBool()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithCreateAPNSProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.CreateAPNSProvider(provID, name, opts...)

	case "fcm":
		var opts []messaging.CreateFCMProviderOption
		if v := plan.ServiceAccount; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateFCMProviderServiceAccountJSON(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithCreateFCMProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.CreateFCMProvider(provID, name, opts...)

	default:
		resp.Diagnostics.AddError("Unsupported provider type", fmt.Sprintf("Provider type %q is not supported.", providerType))
		return
	}

	if err != nil {
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
	messagingClient := appwrite.NewMessaging(r.clients.ClientForProject(projectID))

	id := plan.ID.ValueString()
	providerType := plan.Type.ValueString()

	var prov *models.Provider

	switch providerType {
	case "sendgrid":
		var opts []messaging.UpdateSendgridProviderOption
		if v := plan.Name; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSendgridProviderName(v.ValueString()))
		}
		if v := plan.APIKey; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSendgridProviderApiKey(v.ValueString()))
		}
		if v := plan.FromEmail; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSendgridProviderFromEmail(v.ValueString()))
		}
		if v := plan.FromName; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSendgridProviderFromName(v.ValueString()))
		}
		if v := plan.ReplyToEmail; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSendgridProviderReplyToEmail(v.ValueString()))
		}
		if v := plan.ReplyToName; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSendgridProviderReplyToName(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithUpdateSendgridProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.UpdateSendgridProvider(id, opts...)

	case "mailgun":
		var opts []messaging.UpdateMailgunProviderOption
		if v := plan.Name; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateMailgunProviderName(v.ValueString()))
		}
		if v := plan.APIKey; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateMailgunProviderApiKey(v.ValueString()))
		}
		if v := plan.Domain; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateMailgunProviderDomain(v.ValueString()))
		}
		if v := plan.IsEuRegion; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateMailgunProviderIsEuRegion(v.ValueBool()))
		}
		if v := plan.FromEmail; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateMailgunProviderFromEmail(v.ValueString()))
		}
		if v := plan.FromName; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateMailgunProviderFromName(v.ValueString()))
		}
		if v := plan.ReplyToEmail; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateMailgunProviderReplyToEmail(v.ValueString()))
		}
		if v := plan.ReplyToName; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateMailgunProviderReplyToName(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithUpdateMailgunProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.UpdateMailgunProvider(id, opts...)

	case "smtp":
		var opts []messaging.UpdateSMTPProviderOption
		if v := plan.Name; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSMTPProviderName(v.ValueString()))
		}
		if v := plan.Host; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSMTPProviderHost(v.ValueString()))
		}
		if v := plan.Port; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSMTPProviderPort(int(v.ValueInt64())))
		}
		if v := plan.Username; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSMTPProviderUsername(v.ValueString()))
		}
		if v := plan.Password; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSMTPProviderPassword(v.ValueString()))
		}
		if v := plan.Encryption; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSMTPProviderEncryption(v.ValueString()))
		}
		if v := plan.AutoTLS; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSMTPProviderAutoTLS(v.ValueBool()))
		}
		if v := plan.FromEmail; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSMTPProviderFromEmail(v.ValueString()))
		}
		if v := plan.FromName; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSMTPProviderFromName(v.ValueString()))
		}
		if v := plan.ReplyToEmail; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSMTPProviderReplyToEmail(v.ValueString()))
		}
		if v := plan.ReplyToName; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSMTPProviderReplyToName(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithUpdateSMTPProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.UpdateSMTPProvider(id, opts...)

	case "resend":
		var opts []messaging.UpdateResendProviderOption
		if v := plan.Name; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateResendProviderName(v.ValueString()))
		}
		if v := plan.APIKey; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateResendProviderApiKey(v.ValueString()))
		}
		if v := plan.FromEmail; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateResendProviderFromEmail(v.ValueString()))
		}
		if v := plan.FromName; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateResendProviderFromName(v.ValueString()))
		}
		if v := plan.ReplyToEmail; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateResendProviderReplyToEmail(v.ValueString()))
		}
		if v := plan.ReplyToName; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateResendProviderReplyToName(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithUpdateResendProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.UpdateResendProvider(id, opts...)

	case "twilio":
		var opts []messaging.UpdateTwilioProviderOption
		if v := plan.Name; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateTwilioProviderName(v.ValueString()))
		}
		if v := plan.AccountSid; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateTwilioProviderAccountSid(v.ValueString()))
		}
		if v := plan.AuthToken; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateTwilioProviderAuthToken(v.ValueString()))
		}
		if v := plan.From; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateTwilioProviderFrom(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithUpdateTwilioProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.UpdateTwilioProvider(id, opts...)

	case "vonage":
		var opts []messaging.UpdateVonageProviderOption
		if v := plan.Name; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateVonageProviderName(v.ValueString()))
		}
		if v := plan.APIKey; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateVonageProviderApiKey(v.ValueString()))
		}
		if v := plan.APISecret; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateVonageProviderApiSecret(v.ValueString()))
		}
		if v := plan.From; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateVonageProviderFrom(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithUpdateVonageProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.UpdateVonageProvider(id, opts...)

	case "msg91":
		var opts []messaging.UpdateMsg91ProviderOption
		if v := plan.Name; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateMsg91ProviderName(v.ValueString()))
		}
		if v := plan.SenderID; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateMsg91ProviderSenderId(v.ValueString()))
		}
		if v := plan.AuthKey; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateMsg91ProviderAuthKey(v.ValueString()))
		}
		if v := plan.TemplateID; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateMsg91ProviderTemplateId(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithUpdateMsg91ProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.UpdateMsg91Provider(id, opts...)

	case "telesign":
		var opts []messaging.UpdateTelesignProviderOption
		if v := plan.Name; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateTelesignProviderName(v.ValueString()))
		}
		if v := plan.CustomerID; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateTelesignProviderCustomerId(v.ValueString()))
		}
		if v := plan.APIKey; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateTelesignProviderApiKey(v.ValueString()))
		}
		if v := plan.From; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateTelesignProviderFrom(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithUpdateTelesignProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.UpdateTelesignProvider(id, opts...)

	case "textmagic":
		var opts []messaging.UpdateTextmagicProviderOption
		if v := plan.Name; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateTextmagicProviderName(v.ValueString()))
		}
		if v := plan.Username; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateTextmagicProviderUsername(v.ValueString()))
		}
		if v := plan.APIKey; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateTextmagicProviderApiKey(v.ValueString()))
		}
		if v := plan.From; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateTextmagicProviderFrom(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithUpdateTextmagicProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.UpdateTextmagicProvider(id, opts...)

	case "apns":
		var opts []messaging.UpdateAPNSProviderOption
		if v := plan.Name; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateAPNSProviderName(v.ValueString()))
		}
		if v := plan.AuthKey; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateAPNSProviderAuthKey(v.ValueString()))
		}
		if v := plan.AuthKeyID; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateAPNSProviderAuthKeyId(v.ValueString()))
		}
		if v := plan.TeamID; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateAPNSProviderTeamId(v.ValueString()))
		}
		if v := plan.BundleID; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateAPNSProviderBundleId(v.ValueString()))
		}
		if v := plan.Sandbox; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateAPNSProviderSandbox(v.ValueBool()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithUpdateAPNSProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.UpdateAPNSProvider(id, opts...)

	case "fcm":
		var opts []messaging.UpdateFCMProviderOption
		if v := plan.Name; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateFCMProviderName(v.ValueString()))
		}
		if v := plan.ServiceAccount; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateFCMProviderServiceAccountJSON(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithUpdateFCMProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.UpdateFCMProvider(id, opts...)
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
	// Provider types (sendgrid, smtp, twilio, etc.) match between API and schema.
	if model.Type.IsNull() || model.Type.IsUnknown() {
		model.Type = types.StringValue(prov.Type)
	}
}
