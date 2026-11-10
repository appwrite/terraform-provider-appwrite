package analytics_test

import (
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// The analytics API is served only by Appwrite Cloud; self-hosted Appwrite
// answers its routes with a 404 page. This test stays off unless
// APPWRITE_CLOUD_TESTS is set.
func TestAccAnalyticsPropertyResource_basic(t *testing.T) {
	acceptance.ResourceTest(t, resource.TestCase{
		PreCheck:                 func() { acceptance.CloudPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_analytics_property" "test" {
  name = "tf-acc-test-property"
  type = "mobile"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("appwrite_analytics_property.test", "id"),
					resource.TestCheckResourceAttr("appwrite_analytics_property.test", "name", "tf-acc-test-property"),
					resource.TestCheckResourceAttr("appwrite_analytics_property.test", "type", "mobile"),
					resource.TestCheckResourceAttr("appwrite_analytics_property.test", "domain", ""),
					resource.TestCheckResourceAttr("appwrite_analytics_property.test", "enabled", "true"),
					resource.TestCheckResourceAttr("appwrite_analytics_property.test", "identify_users", "false"),
					resource.TestCheckResourceAttr("appwrite_analytics_property.test", "allowed_origins.#", "1"),
					resource.TestCheckResourceAttr("appwrite_analytics_property.test", "allowed_origins.0", "*"),
				),
			},
			{
				ResourceName:      "appwrite_analytics_property.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: `
resource "appwrite_analytics_property" "test" {
  name            = "tf-acc-test-property-updated"
  type            = "web"
  domain          = "example.com"
  enabled         = false
  allowed_origins = ["https://example.com", "https://www.example.com"]
  identify_users  = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_analytics_property.test", "name", "tf-acc-test-property-updated"),
					resource.TestCheckResourceAttr("appwrite_analytics_property.test", "type", "web"),
					resource.TestCheckResourceAttr("appwrite_analytics_property.test", "domain", "example.com"),
					resource.TestCheckResourceAttr("appwrite_analytics_property.test", "enabled", "false"),
					resource.TestCheckResourceAttr("appwrite_analytics_property.test", "identify_users", "true"),
					resource.TestCheckResourceAttr("appwrite_analytics_property.test", "allowed_origins.#", "2"),
				),
			},
			{
				// Leaving the optional attributes out puts each back to its default,
				// which is what clears the domain.
				Config: `
resource "appwrite_analytics_property" "test" {
  name = "tf-acc-test-property"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_analytics_property.test", "domain", ""),
					resource.TestCheckResourceAttr("appwrite_analytics_property.test", "enabled", "true"),
					resource.TestCheckResourceAttr("appwrite_analytics_property.test", "allowed_origins.0", "*"),
				),
			},
		},
	})
}
