package project

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/sdk-for-go/v7/project"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// emailTemplateTypes are the template types Appwrite 2.2.0 accepts, from
// app/config/locale/templates.php.
var emailTemplateTypes = []string{
	"verification",
	"magicSession",
	"recovery",
	"invitation",
	"mfaChallenge",
	"sessionAlert",
	"otpSession",
}

// emailTemplateLocales are the locale codes Appwrite 2.2.0 accepts for a
// template, from app/config/locale/codes.php. Appwrite falls back to English
// text for any of them it has no translation for.
var emailTemplateLocales = []string{
	"af", "ar-ae", "ar-bh", "ar-dz", "ar-eg", "ar-iq", "ar-jo", "ar-kw", "ar-lb", "ar-ly",
	"ar-ma", "ar-om", "ar-qa", "ar-sa", "ar-sy", "ar-tn", "ar-ye", "as", "az", "be",
	"bg", "bh", "bn", "bs", "ca", "cs", "cy", "da", "de", "de-at",
	"de-ch", "de-li", "de-lu", "el", "en", "en-au", "en-bz", "en-ca", "en-gb", "en-ie",
	"en-jm", "en-nz", "en-tt", "en-us", "en-za", "eo", "es", "es-ar", "es-bo", "es-cl",
	"es-co", "es-cr", "es-do", "es-ec", "es-gt", "es-hn", "es-mx", "es-ni", "es-pa", "es-pe",
	"es-pr", "es-py", "es-sv", "es-uy", "es-ve", "et", "eu", "fa", "fi", "fo",
	"fr", "fr-be", "fr-ca", "fr-ch", "fr-lu", "ga", "gd", "he", "hi", "hr",
	"hu", "id", "is", "it", "it-ch", "ja", "ji", "ko", "ku", "lt",
	"lv", "mk", "ml", "ms", "mt", "nb", "ne", "nl", "nl-be", "nn",
	"no", "pa", "pl", "pt", "pt-br", "rm", "ro", "ro-md", "ru", "ru-md",
	"sb", "sk", "sl", "sq", "sr", "sv", "sv-fi", "th", "tn", "tr",
	"ts", "ua", "ur", "ve", "vi", "xh", "zh-cn", "zh-hk", "zh-sg", "zh-tw",
	"zu",
}

// smtpSecureValues are the encryption modes the SMTP route accepts. There is
// no value for "none": once one of these is stored, the API cannot clear it.
var smtpSecureValues = []string{"tls", "ssl"}

const (
	smtpResourceName     = "appwrite_project_smtp"
	templateResourceName = "appwrite_project_email_template"
)

var (
	smtpScopes     = []string{"project.read", "project.write"}
	templateScopes = []string{"templates.read", "templates.write"}
)

// projectSettingsClient returns the Project service for a project-scoped
// settings resource, after rejecting credentials that cannot reach it.
func projectSettingsClient(clients *common.AppwriteClients, resourceName, projectID string, scopes ...string) (*project.Project, error) {
	if err := common.ValidateProjectCredential(clients, resourceName, scopes...); err != nil {
		return nil, err
	}
	return appwrite.NewProject(clients.ClientForProject(projectID)), nil
}

// hostnamePattern mirrors the API's Hostname validator, which rejects a scheme,
// a path or a port.
var hostnamePattern = regexp.MustCompile(`^[^/:]+$`)

func smtpHostValidators() []validator.String {
	return []validator.String{
		stringvalidator.LengthBetween(1, 253),
		stringvalidator.RegexMatches(hostnamePattern, "must be a host name or IP address, without a scheme, path or port"),
	}
}

// smtpSettings is what an SMTP update sends. Every field is sent on every
// update, so a value removed from the configuration is cleared on the server
// rather than left as it was.
type smtpSettings struct {
	Enabled      bool
	Host         string
	Port         int
	Username     string
	Password     string
	SenderEmail  string
	SenderName   string
	ReplyToEmail string
	ReplyToName  string
	// Secure is omitted from the request when empty. The API has no value for
	// "no encryption", only tls and ssl, so an empty string is rejected.
	Secure string
}

func updateSMTP(client *project.Project, settings smtpSettings) (*models.Project, error) {
	opts := []project.UpdateSMTPOption{
		client.WithUpdateSMTPEnabled(settings.Enabled),
		client.WithUpdateSMTPHost(settings.Host),
		client.WithUpdateSMTPPort(settings.Port),
		client.WithUpdateSMTPUsername(settings.Username),
		client.WithUpdateSMTPPassword(settings.Password),
		client.WithUpdateSMTPSenderEmail(settings.SenderEmail),
		client.WithUpdateSMTPSenderName(settings.SenderName),
		client.WithUpdateSMTPReplyToEmail(settings.ReplyToEmail),
		client.WithUpdateSMTPReplyToName(settings.ReplyToName),
	}
	if settings.Secure != "" {
		opts = append(opts, client.WithUpdateSMTPSecure(settings.Secure))
	}
	return client.UpdateSMTP(opts...)
}

// resetSMTP disables custom SMTP and clears every field the API lets a caller
// clear. Disabling explicitly also stops Appwrite from connecting to the SMTP
// server to check it, so a reset succeeds even when that server is gone.
//
// The host, port and encryption mode stay: the API rejects an empty host and
// has no value meaning "no encryption". With custom SMTP disabled Appwrite
// ignores all three, and custom email templates stop being used too.
func resetSMTP(client *project.Project) (*models.Project, error) {
	return client.UpdateSMTP(
		client.WithUpdateSMTPEnabled(false),
		client.WithUpdateSMTPUsername(""),
		client.WithUpdateSMTPPassword(""),
		client.WithUpdateSMTPSenderEmail(""),
		client.WithUpdateSMTPSenderName(""),
		client.WithUpdateSMTPReplyToEmail(""),
		client.WithUpdateSMTPReplyToName(""),
	)
}

// smtpConfigured reports whether a project has any custom SMTP configuration
// to manage. A project that never had one has no host, and neither does one
// whose configuration was never saved.
func smtpConfigured(p *models.Project) bool {
	return p.SmtpEnabled || p.SmtpHost != ""
}

// emailTemplateSettings is what a template update sends. As with SMTP, every
// field is sent, so removing an optional one from the configuration clears it.
type emailTemplateSettings struct {
	Type         string
	Locale       string
	Subject      string
	Message      string
	SenderName   string
	SenderEmail  string
	ReplyToEmail string
	ReplyToName  string
}

func updateEmailTemplate(client *project.Project, settings emailTemplateSettings) (*models.EmailTemplate, error) {
	return client.UpdateEmailTemplate(
		settings.Type,
		client.WithUpdateEmailTemplateLocale(settings.Locale),
		client.WithUpdateEmailTemplateSubject(settings.Subject),
		client.WithUpdateEmailTemplateMessage(settings.Message),
		client.WithUpdateEmailTemplateSenderName(settings.SenderName),
		client.WithUpdateEmailTemplateSenderEmail(settings.SenderEmail),
		client.WithUpdateEmailTemplateReplyToEmail(settings.ReplyToEmail),
		client.WithUpdateEmailTemplateReplyToName(settings.ReplyToName),
	)
}

// findCustomEmailTemplate returns the project's custom template for a type and
// locale, or nil when the project uses Appwrite's default for it.
//
// The list is the only way to tell the two apart. Getting a single template
// returns Appwrite's built-in text when there is no custom one, so it always
// succeeds and cannot show that a template was reset outside Terraform. The
// list holds custom templates only, and without a limit query it returns all
// of them.
func findCustomEmailTemplate(client *project.Project, templateType, locale string) (*models.EmailTemplate, error) {
	list, err := client.ListEmailTemplates()
	if err != nil {
		return nil, err
	}
	for i := range list.Templates {
		template := list.Templates[i]
		if template.TemplateId == templateType && template.Locale == locale {
			return &template, nil
		}
	}
	return nil, nil
}

func emailTemplateID(templateType, locale string) string {
	return templateType + "/" + locale
}

// parseEmailTemplateImportID accepts type/locale, or project_id/type/locale.
func parseEmailTemplateImportID(id string) (projectID, templateType, locale string, err error) {
	parts := strings.Split(id, "/")
	for _, part := range parts {
		if part == "" {
			parts = nil
			break
		}
	}
	switch len(parts) {
	case 2:
		return "", parts[0], parts[1], nil
	case 3:
		return parts[0], parts[1], parts[2], nil
	default:
		return "", "", "", fmt.Errorf("expected type/locale or project_id/type/locale, got: %q", id)
	}
}

// optionalString maps an API string onto an optional attribute. The API
// returns an empty string for a field that was never set, and keeping that as
// null when the configuration leaves the attribute out stops it planning a
// change from null to "".
func optionalString(prior types.String, actual string) types.String {
	if actual == "" && prior.IsNull() {
		return types.StringNull()
	}
	return types.StringValue(actual)
}
