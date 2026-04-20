package webhook_test

import (
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWebhookDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_webhook" "test_ds" {
  name   = "DS Test Webhook"
  url    = "https://example.com/webhook"
  events = ["databases.*.collections.*.documents.*.create"]
}

data "appwrite_webhook" "test" {
  id = appwrite_webhook.test_ds.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.appwrite_webhook.test", "name", "DS Test Webhook"),
					resource.TestCheckResourceAttr("data.appwrite_webhook.test", "url", "https://example.com/webhook"),
					resource.TestCheckResourceAttrSet("data.appwrite_webhook.test", "created_at"),
				),
			},
		},
	})
}
