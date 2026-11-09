package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/appwrite/sdk-for-go/v7/client"
	"github.com/appwrite/sdk-for-go/v7/messaging"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Provider types that other code here refers to by name.
const (
	typeSES      = "ses"
	typeAppwrite = "appwrite"
)

// errUnsupportedType is returned for a provider type with no create or update
// route here.
var errUnsupportedType = errors.New("unsupported provider type")

// errSMTPHostRequired is returned when an smtp provider has no host, the one
// argument its create route requires.
var errSMTPHostRequired = errors.New("host is required for smtp providers")

// createProvider sends plan to the create route for its type. Each type has
// its own route and its own SDK call.
func createProvider(messagingClient *messaging.Messaging, provID string, plan providerResourceModel) (*models.Provider, error) {
	name := plan.Name.ValueString()
	providerType := plan.Type.ValueString()

	var prov *models.Provider
	var err error

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
			return nil, errSMTPHostRequired
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

	case typeSES:
		var opts []messaging.CreateSesProviderOption
		if v := plan.AccessKey; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateSesProviderAccessKey(v.ValueString()))
		}
		if v := plan.SecretKey; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateSesProviderSecretKey(v.ValueString()))
		}
		if v := plan.Region; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateSesProviderRegion(v.ValueString()))
		}
		if v := plan.FromEmail; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateSesProviderFromEmail(v.ValueString()))
		}
		if v := plan.FromName; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateSesProviderFromName(v.ValueString()))
		}
		if v := plan.ReplyToEmail; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateSesProviderReplyToEmail(v.ValueString()))
		}
		if v := plan.ReplyToName; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateSesProviderReplyToName(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithCreateSesProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.CreateSesProvider(provID, name, opts...)

	case typeAppwrite:
		var opts []messaging.CreateAppwriteProviderOption
		if v := plan.Qos; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateAppwriteProviderQos(int(v.ValueInt64())))
		}
		if v := plan.Expiry; !v.IsNull() {
			opts = append(opts, messagingClient.WithCreateAppwriteProviderExpiry(int(v.ValueInt64())))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithCreateAppwriteProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.CreateAppwriteProvider(provID, name, opts...)

	default:
		return nil, fmt.Errorf("%w %q", errUnsupportedType, providerType)
	}

	return prov, err
}

// updateProvider sends plan to the update route for its type. c is the client
// messagingClient wraps, for the one route the SDK cannot express.
func updateProvider(c client.Client, messagingClient *messaging.Messaging, plan providerResourceModel) (*models.Provider, error) {
	id := plan.ID.ValueString()
	providerType := plan.Type.ValueString()

	var prov *models.Provider
	var err error

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

	case typeSES:
		var opts []messaging.UpdateSesProviderOption
		if v := plan.Name; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSesProviderName(v.ValueString()))
		}
		if v := plan.AccessKey; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSesProviderAccessKey(v.ValueString()))
		}
		if v := plan.SecretKey; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSesProviderSecretKey(v.ValueString()))
		}
		if v := plan.Region; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSesProviderRegion(v.ValueString()))
		}
		if v := plan.FromEmail; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSesProviderFromEmail(v.ValueString()))
		}
		if v := plan.FromName; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSesProviderFromName(v.ValueString()))
		}
		if v := plan.ReplyToEmail; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSesProviderReplyToEmail(v.ValueString()))
		}
		if v := plan.ReplyToName; !v.IsNull() {
			opts = append(opts, messagingClient.WithUpdateSesProviderReplyToName(v.ValueString()))
		}
		if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
			opts = append(opts, messagingClient.WithUpdateSesProviderEnabled(v.ValueBool()))
		}
		prov, err = messagingClient.UpdateSesProvider(id, opts...)

	case typeAppwrite:
		prov, err = updateAppwriteProvider(c, plan)

	default:
		return nil, fmt.Errorf("%w %q", errUnsupportedType, providerType)
	}

	return prov, err
}

// updateAppwriteProvider updates an Appwrite push provider with a raw call.
//
// qos and expiry are nullable on the server, and an explicit null resets
// them: subscribers choose the QoS, and topics fall back to their own expiry.
// An omitted one is left as it was. The SDK takes both as int, so it can send
// a value or leave the field out but cannot send null. Sending both on every
// update, null when the plan leaves them unset, is what makes removing one
// from the configuration actually clear it.
func updateAppwriteProvider(c client.Client, plan providerResourceModel) (*models.Provider, error) {
	params := map[string]interface{}{
		"qos":    int64OrNil(plan.Qos),
		"expiry": int64OrNil(plan.Expiry),
	}
	if v := plan.Name; !v.IsNull() {
		params["name"] = v.ValueString()
	}
	if v := plan.Enabled; !v.IsNull() && !v.IsUnknown() {
		params["enabled"] = v.ValueBool()
	}

	path := "/messaging/providers/appwrite/" + client.EncodePath(plan.ID.ValueString())
	resp, err := c.Call("PATCH", path, map[string]interface{}{
		// client.Call does not add the project header; see common.GetColumnRawJSON.
		"X-Appwrite-Project": c.Config["project"],
		"content-type":       "application/json",
		"accept":             "application/json",
	}, params)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(resp.Type, "application/json") {
		return nil, fmt.Errorf("unexpected response type: %s", resp.Type)
	}
	body, err := client.ResponseBody(resp)
	if err != nil {
		return nil, err
	}
	prov := &models.Provider{}
	if err := json.Unmarshal(body, prov); err != nil {
		return nil, fmt.Errorf("decoding provider: %w", err)
	}
	return prov, nil
}

// isRouteMissing reports whether err is a 404 for a route the server does not
// have, rather than for a resource it could not find. Appwrite answers an
// unknown route with an HTML error page, so the response carries no error
// type.
func isRouteMissing(err error) bool {
	return common.IsNotFoundError(err) && common.ErrorType(err) == ""
}

func int64OrNil(v types.Int64) interface{} {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	return v.ValueInt64()
}
