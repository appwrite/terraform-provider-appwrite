package database_test

import (
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDatabaseDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_database" "test" {
  id   = "analytics"
  name = "Analytics"
}

data "appwrite_database" "test" {
  id = appwrite_database.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.appwrite_database.test", "name", "Analytics"),
					resource.TestCheckResourceAttr("data.appwrite_database.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("data.appwrite_database.test", "created_at"),
				),
			},
		},
	})
}
