package site_test

import (
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSiteVariableResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_site" "test" {
  name          = "variable-test-site"
  framework     = "other"
  build_runtime = "node-22"
}

resource "appwrite_site_variable" "test" {
  site_id = appwrite_site.test.id
  key     = "API_URL"
  value   = "https://api.example.com"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("appwrite_site_variable.test", "id"),
					resource.TestCheckResourceAttr("appwrite_site_variable.test", "key", "API_URL"),
					resource.TestCheckResourceAttrSet("appwrite_site_variable.test", "created_at"),
				),
			},
			{
				Config: `
resource "appwrite_site" "test" {
  name          = "variable-test-site"
  framework     = "other"
  build_runtime = "node-22"
}

resource "appwrite_site_variable" "test" {
  site_id = appwrite_site.test.id
  key     = "API_URL"
  value   = "https://api.example.com/v2"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_site_variable.test", "key", "API_URL"),
				),
			},
		},
	})
}
