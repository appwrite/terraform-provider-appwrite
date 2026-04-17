package webhook_test

import (
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWebhookResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_webhook" "test" {
  name   = "Order Webhook"
  url    = "https://example.com/webhook"
  events = ["databases.*.collections.*.documents.*.create"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("appwrite_webhook.test", "id"),
					resource.TestCheckResourceAttr("appwrite_webhook.test", "name", "Order Webhook"),
					resource.TestCheckResourceAttr("appwrite_webhook.test", "url", "https://example.com/webhook"),
					resource.TestCheckResourceAttr("appwrite_webhook.test", "enabled", "true"),
					resource.TestCheckResourceAttr("appwrite_webhook.test", "events.#", "1"),
					resource.TestCheckResourceAttrSet("appwrite_webhook.test", "secret"),
				),
			},
			{
				ResourceName:            "appwrite_webhook.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"auth_password"},
			},
			{
				Config: `
resource "appwrite_webhook" "test" {
  name   = "Updated Webhook"
  url    = "https://example.com/webhook/v2"
  events = ["databases.*.collections.*.documents.*.create", "databases.*.collections.*.documents.*.update"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_webhook.test", "name", "Updated Webhook"),
					resource.TestCheckResourceAttr("appwrite_webhook.test", "events.#", "2"),
				),
			},
		},
	})
}
