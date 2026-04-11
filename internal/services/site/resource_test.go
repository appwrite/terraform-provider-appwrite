package site_test

import (
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSiteResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_site" "test" {
  name          = "test-site"
  framework     = "other"
  build_runtime = "node-22"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("appwrite_site.test", "id"),
					resource.TestCheckResourceAttr("appwrite_site.test", "name", "test-site"),
					resource.TestCheckResourceAttr("appwrite_site.test", "framework", "other"),
					resource.TestCheckResourceAttr("appwrite_site.test", "build_runtime", "node-22"),
					resource.TestCheckResourceAttr("appwrite_site.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("appwrite_site.test", "created_at"),
				),
			},
			{
				ResourceName:      "appwrite_site.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: `
resource "appwrite_site" "test" {
  name          = "updated-site"
  framework     = "other"
  build_runtime = "node-22"
  enabled       = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_site.test", "name", "updated-site"),
					resource.TestCheckResourceAttr("appwrite_site.test", "enabled", "false"),
				),
			},
		},
	})
}

func TestAccSiteResource_nextjs(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_site" "test_nextjs" {
  name            = "nextjs-site"
  framework       = "nextjs"
  build_runtime   = "node-22"
  install_command = "npm install"
  build_command   = "npm run build"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("appwrite_site.test_nextjs", "id"),
					resource.TestCheckResourceAttr("appwrite_site.test_nextjs", "framework", "nextjs"),
					resource.TestCheckResourceAttr("appwrite_site.test_nextjs", "install_command", "npm install"),
					resource.TestCheckResourceAttr("appwrite_site.test_nextjs", "build_command", "npm run build"),
				),
			},
		},
	})
}
