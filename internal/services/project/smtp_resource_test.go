package project_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// Both tests in this file change the same per-project setting, so neither runs
// in parallel.

// With enabled = false Appwrite saves the configuration without connecting to
// the server, so this runs against a host that does not exist. It covers every
// field, clearing optional ones by removing them, switching to the write-only
// password, import, and destroy resetting the configuration.
func TestAccProjectSMTP_disabled(t *testing.T) {
	const name = "appwrite_project_smtp.test"
	projectID := os.Getenv("APPWRITE_PROJECT_ID")

	full := `
resource "appwrite_project_smtp" "test" {
  enabled        = false
  host           = "smtp.tf-acc-test.invalid"
  port           = 2525
  username       = "tf-acc-user"
  password       = "tf-acc-password"
  sender_email   = "noreply@tf-acc-test.example.com"
  sender_name    = "tf-acc-test sender"
  reply_to_email = "support@tf-acc-test.example.com"
  reply_to_name  = "tf-acc-test support"
}
`
	trimmed := `
resource "appwrite_project_smtp" "test" {
  enabled             = false
  host                = "smtp2.tf-acc-test.invalid"
  port                = 587
  username            = "tf-acc-user-2"
  password_wo         = "tf-acc-password-2"
  password_wo_version = 1
  sender_email        = "noreply@tf-acc-test.example.com"
}
`

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_11_0),
		},
		Steps: []resource.TestStep{
			{
				Config: full,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "id", projectID),
					resource.TestCheckResourceAttr(name, "project_id", projectID),
					resource.TestCheckResourceAttr(name, "enabled", "false"),
					resource.TestCheckResourceAttr(name, "host", "smtp.tf-acc-test.invalid"),
					resource.TestCheckResourceAttr(name, "port", "2525"),
					resource.TestCheckResourceAttr(name, "password", "tf-acc-password"),
					resource.TestCheckNoResourceAttr(name, "secure"),
					checkSMTPOnServer(t, func(p *models.Project) error {
						return expectFields(map[string]string{
							"host":           p.SmtpHost,
							"username":       p.SmtpUsername,
							"sender_email":   p.SmtpSenderEmail,
							"sender_name":    p.SmtpSenderName,
							"reply_to_email": p.SmtpReplyToEmail,
							"reply_to_name":  p.SmtpReplyToName,
						}, map[string]string{
							"host":           "smtp.tf-acc-test.invalid",
							"username":       "tf-acc-user",
							"sender_email":   "noreply@tf-acc-test.example.com",
							"sender_name":    "tf-acc-test sender",
							"reply_to_email": "support@tf-acc-test.example.com",
							"reply_to_name":  "tf-acc-test support",
						})
					}),
				),
			},
			{
				// Removing the optional fields clears them on the server, and
				// the write-only password keeps the secret out of state.
				Config: trimmed,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "host", "smtp2.tf-acc-test.invalid"),
					resource.TestCheckResourceAttr(name, "port", "587"),
					resource.TestCheckNoResourceAttr(name, "password"),
					resource.TestCheckNoResourceAttr(name, "password_wo"),
					resource.TestCheckNoResourceAttr(name, "sender_name"),
					resource.TestCheckNoResourceAttr(name, "reply_to_email"),
					resource.TestCheckNoResourceAttr(name, "reply_to_name"),
					checkSMTPOnServer(t, func(p *models.Project) error {
						return expectFields(map[string]string{
							"username":       p.SmtpUsername,
							"sender_name":    p.SmtpSenderName,
							"reply_to_email": p.SmtpReplyToEmail,
							"reply_to_name":  p.SmtpReplyToName,
						}, map[string]string{
							"username": "tf-acc-user-2",
						})
					}),
				),
			},
			acceptance.ExpectEmptyPlanAfterApply(trimmed),
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateId:     projectID,
				ImportStateVerify: true,
				// Appwrite never returns the password, and the write-only
				// argument and its version exist only in configuration.
				ImportStateVerifyIgnore: []string{"password", "password_wo", "password_wo_version"},
			},
			{
				// Appwrite refuses custom templates while custom SMTP is
				// disabled. The error has to reach the user as is.
				Config: trimmed + `
resource "appwrite_project_email_template" "test" {
  type    = "verification"
  locale  = "en"
  subject = "tf-acc-test subject"
  message = "tf-acc-test message"

  depends_on = [appwrite_project_smtp.test]
}
`,
				ExpectError: regexp.MustCompile(`SMTP must be enabled`),
			},
		},
	})
}

// Enabling custom SMTP makes Appwrite connect to the server, so this test runs
// a stub one (see acceptance.SMTPServer). Custom templates need custom SMTP
// enabled, so they are covered here too.
func TestAccProjectSMTP_enabledWithTemplates(t *testing.T) {
	host, port := acceptance.SMTPServer(t)
	projectID := os.Getenv("APPWRITE_PROJECT_ID")

	config := func(verificationMessage, verificationExtra string) string {
		return fmt.Sprintf(`
resource "appwrite_project_smtp" "test" {
  host                = %q
  port                = %d
  username            = "tf-acc-user"
  password_wo         = "tf-acc-password"
  password_wo_version = 1
  sender_email        = "noreply@tf-acc-test.example.com"
  sender_name         = "tf-acc-test"
}

resource "appwrite_project_email_template" "verification" {
  type    = "verification"
  locale  = "en"
  subject = "Verify your tf-acc-test account"
  message = %q
%s
  depends_on = [appwrite_project_smtp.test]
}

resource "appwrite_project_email_template" "recovery" {
  type           = "recovery"
  locale         = "pt-br"
  subject        = "Recupere sua conta tf-acc-test"
  message        = "<p>Olá {{user}}, use {{redirect}}.</p>"
  sender_email   = "recovery@tf-acc-test.example.com"
  reply_to_email = "support@tf-acc-test.example.com"
  reply_to_name  = "tf-acc-test support"

  depends_on = [appwrite_project_smtp.test]
}
`, host, port, verificationMessage, verificationExtra)
	}

	initial := config("<p>Hello {{user}}, verify at {{redirect}}.</p>", `  sender_name = "tf-acc-test verification"`)
	updated := config("<p>Hi {{user}}, verify your address: {{redirect}}</p>", "")

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_11_0),
		},
		// The registered checks show custom SMTP was reset. The templates
		// cannot be removed, so this pins that destroy left them as they were
		// rather than leaving that unchecked.
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			acceptance.CheckDestroy(t),
			checkTemplateOnServer(t, "verification", "en", "<p>Hi {{user}}, verify your address: {{redirect}}</p>"),
		),
		Steps: []resource.TestStep{
			{
				Config: initial,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_project_smtp.test", "enabled", "true"),
					resource.TestCheckResourceAttr("appwrite_project_smtp.test", "username", "tf-acc-user"),
					resource.TestCheckNoResourceAttr("appwrite_project_smtp.test", "password"),
					checkSMTPOnServer(t, func(p *models.Project) error {
						if !p.SmtpEnabled {
							return fmt.Errorf("custom SMTP is not enabled")
						}
						return nil
					}),
					resource.TestCheckResourceAttr("appwrite_project_email_template.verification", "id", "verification/en"),
					resource.TestCheckResourceAttr("appwrite_project_email_template.verification", "sender_name", "tf-acc-test verification"),
					resource.TestCheckNoResourceAttr("appwrite_project_email_template.verification", "sender_email"),
					resource.TestCheckResourceAttr("appwrite_project_email_template.recovery", "id", "recovery/pt-br"),
					resource.TestCheckResourceAttr("appwrite_project_email_template.recovery", "message", "<p>Olá {{user}}, use {{redirect}}.</p>"),
					checkTemplateOnServer(t, "recovery", "pt-br", "<p>Olá {{user}}, use {{redirect}}.</p>"),
				),
			},
			{
				Config: updated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_project_email_template.verification", "message", "<p>Hi {{user}}, verify your address: {{redirect}}</p>"),
					resource.TestCheckNoResourceAttr("appwrite_project_email_template.verification", "sender_name"),
					checkTemplateOnServer(t, "verification", "en", "<p>Hi {{user}}, verify your address: {{redirect}}</p>"),
					checkTemplateSenderName(t, "verification", "en", ""),
				),
			},
			acceptance.ExpectEmptyPlanAfterApply(updated),
			{
				ResourceName:      "appwrite_project_email_template.recovery",
				ImportState:       true,
				ImportStateId:     "recovery/pt-br",
				ImportStateVerify: true,
			},
			{
				ResourceName:      "appwrite_project_email_template.verification",
				ImportState:       true,
				ImportStateId:     projectID + "/verification/en",
				ImportStateVerify: true,
			},
			{
				ResourceName:            "appwrite_project_smtp.test",
				ImportState:             true,
				ImportStateId:           projectID,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password", "password_wo", "password_wo_version"},
			},
		},
	})
}

func checkSMTPOnServer(t *testing.T, check func(*models.Project) error) resource.TestCheckFunc {
	t.Helper()
	return func(*terraform.State) error {
		p, err := appwrite.NewProject(acceptance.ProjectClient(t)).Get()
		if err != nil {
			return fmt.Errorf("reading the project: %w", err)
		}
		return check(p)
	}
}

func findTemplate(t *testing.T, templateType, locale string) (*models.EmailTemplate, error) {
	t.Helper()
	list, err := appwrite.NewProject(acceptance.ProjectClient(t)).ListEmailTemplates()
	if err != nil {
		return nil, fmt.Errorf("listing email templates: %w", err)
	}
	for i := range list.Templates {
		if list.Templates[i].TemplateId == templateType && list.Templates[i].Locale == locale {
			return &list.Templates[i], nil
		}
	}
	return nil, fmt.Errorf("no custom %s/%s template on the server", templateType, locale)
}

func checkTemplateOnServer(t *testing.T, templateType, locale, message string) resource.TestCheckFunc {
	t.Helper()
	return func(*terraform.State) error {
		template, err := findTemplate(t, templateType, locale)
		if err != nil {
			return err
		}
		if template.Message != message {
			return fmt.Errorf("%s/%s message is %q, want %q", templateType, locale, template.Message, message)
		}
		return nil
	}
}

func checkTemplateSenderName(t *testing.T, templateType, locale, senderName string) resource.TestCheckFunc {
	t.Helper()
	return func(*terraform.State) error {
		template, err := findTemplate(t, templateType, locale)
		if err != nil {
			return err
		}
		if template.SenderName != senderName {
			return fmt.Errorf("%s/%s sender name is %q, want %q", templateType, locale, template.SenderName, senderName)
		}
		return nil
	}
}

// expectFields compares named values against what they should be; a name
// missing from want must be empty.
func expectFields(got, want map[string]string) error {
	for name, value := range got {
		if value != want[name] {
			return fmt.Errorf("%s is %q on the server, want %q", name, value, want[name])
		}
	}
	return nil
}
