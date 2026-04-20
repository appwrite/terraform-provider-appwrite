package topic_test

import (
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTopicDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_messaging_topic" "test_ds" {
  id   = "ds-test-topic"
  name = "DS Test Topic"
}

data "appwrite_messaging_topic" "test" {
  id = appwrite_messaging_topic.test_ds.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.appwrite_messaging_topic.test", "name", "DS Test Topic"),
					resource.TestCheckResourceAttrSet("data.appwrite_messaging_topic.test", "created_at"),
				),
			},
		},
	})
}
