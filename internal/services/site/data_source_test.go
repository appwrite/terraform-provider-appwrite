package site_test

import (
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSiteDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_site" "test_ds" {
  name          = "DS Test Site"
  framework     = "other"
  build_runtime = "node-22"
}

data "appwrite_site" "test" {
  id = appwrite_site.test_ds.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.appwrite_site.test", "name", "DS Test Site"),
					resource.TestCheckResourceAttr("data.appwrite_site.test", "framework", "other"),
					resource.TestCheckResourceAttr("data.appwrite_site.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("data.appwrite_site.test", "created_at"),
				),
			},
		},
	})
}
