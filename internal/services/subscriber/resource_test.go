package subscriber_test

import (
	"fmt"
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccMessagingSubscriberResource_basic(t *testing.T) {
	// This subscribes the target Appwrite creates alongside a user for their
	// email address, which no resource manages and appwrite_user_target
	// refuses to import; the fixture provisions one through the Appwrite
	// client. Subscribing an appwrite_user_target is covered by
	// TestAccUserTarget_basic. The hardcoded "target-123" this replaces
	// resolved against nothing and failed from the day it was written.
	targetID, cleanup := acceptance.MessagingTarget(t)
	defer cleanup()

	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSubscriberConfig(targetID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("appwrite_messaging_subscriber.test", "id"),
					resource.TestCheckResourceAttr("appwrite_messaging_subscriber.test", "target_id", targetID),
					resource.TestCheckResourceAttrSet("appwrite_messaging_subscriber.test", "created_at"),
				),
			},
			{
				ResourceName: "appwrite_messaging_subscriber.test",
				ImportState:  true,
				// The topic ID is server-generated, so the composite import ID
				// has to be built from state rather than written as a literal.
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["appwrite_messaging_subscriber.test"]
					if !ok {
						return "", fmt.Errorf("resource appwrite_messaging_subscriber.test not found in state")
					}
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes["topic_id"], rs.Primary.Attributes["id"]), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func testAccSubscriberConfig(targetID string) string {
	return fmt.Sprintf(`
resource "appwrite_messaging_topic" "test" {
  name = "subscriber-test-topic"
}

resource "appwrite_messaging_subscriber" "test" {
  topic_id  = appwrite_messaging_topic.test.id
  target_id = %q
}
`, targetID)
}
