package topic_test

import (
	"fmt"
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTopicResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTopicConfig("announcements", "Announcements"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_topic.test", "id", "announcements"),
					resource.TestCheckResourceAttr("appwrite_topic.test", "name", "Announcements"),
					resource.TestCheckResourceAttrSet("appwrite_topic.test", "created_at"),
				),
			},
			{
				ResourceName:      "appwrite_topic.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccTopicConfig("announcements", "Company Announcements"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_topic.test", "name", "Company Announcements"),
				),
			},
		},
	})
}

func TestAccTopicResource_with_subscribe(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_topic" "test" {
  id        = "alerts"
  name      = "Alerts"
  subscribe = ["users"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_topic.test", "subscribe.#", "1"),
					resource.TestCheckResourceAttr("appwrite_topic.test", "subscribe.0", "users"),
				),
			},
		},
	})
}

func testAccTopicConfig(id, name string) string {
	return fmt.Sprintf(`
resource "appwrite_topic" "test" {
  id   = %q
  name = %q
}
`, id, name)
}
