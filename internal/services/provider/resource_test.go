package provider_test

import (
	"fmt"
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMessagingProviderResource_sendgrid(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMessagingProviderSendgrid("sendgrid-test", "Sendgrid"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "id", "sendgrid-test"),
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "name", "Sendgrid"),
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "type", "sendgrid"),
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("appwrite_messaging_provider.test", "created_at"),
				),
			},
			{
				ResourceName:            "appwrite_messaging_provider.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"api_key"},
			},
			{
				Config: testAccMessagingProviderSendgrid("sendgrid-test", "Sendgrid Updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "name", "Sendgrid Updated"),
				),
			},
		},
	})
}

func TestAccMessagingProviderResource_smtp(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_messaging_provider" "test" {
  id         = "smtp-test"
  name       = "SMTP"
  type       = "smtp"
  host       = "smtp.example.com"
  port       = 587
  encryption = "tls"
  from_email = "noreply@example.com"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "type", "smtp"),
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "enabled", "true"),
				),
			},
		},
	})
}

func TestAccMessagingProviderResource_twilio(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_messaging_provider" "test" {
  id   = "twilio-test"
  name = "Twilio"
  type = "twilio"
  from = "+1234567890"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_messaging_provider.test", "type", "twilio"),
				),
			},
		},
	})
}

func testAccMessagingProviderSendgrid(id, name string) string {
	return fmt.Sprintf(`
resource "appwrite_messaging_provider" "test" {
  id         = %q
  name       = %q
  type       = "sendgrid"
  api_key    = "SG.test-key"
  from_email = "noreply@example.com"
  from_name  = "Test App"
}
`, id, name)
}
