package subscriber_test

import (
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMessagingSubscriberResource_basic(t *testing.T) {
	// A subscriber points at a messaging target, and the provider exposes no
	// way to obtain one: targets are created implicitly alongside a user, and
	// neither appwrite_auth_user nor its data source surfaces them. The
	// hardcoded "target-123" below has never resolved against a real Appwrite,
	// so this test failed from the day it was written. Unskip it once the user
	// resource or data source exposes targets.
	t.Skip("no supported way to obtain a messaging target ID; see appwrite-labs/cloud#5914")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_messaging_topic" "test" {
  name = "subscriber-test-topic"
}

resource "appwrite_messaging_subscriber" "test" {
  topic_id  = appwrite_messaging_topic.test.id
  target_id = "target-123"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("appwrite_messaging_subscriber.test", "id"),
					resource.TestCheckResourceAttr("appwrite_messaging_subscriber.test", "target_id", "target-123"),
					resource.TestCheckResourceAttrSet("appwrite_messaging_subscriber.test", "created_at"),
				),
			},
			{
				ResourceName:      "appwrite_messaging_subscriber.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
