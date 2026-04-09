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
resource "appwrite_tablesdb" "test" {
  id   = "analytics"
  name = "Analytics"
}

data "appwrite_tablesdb" "test" {
  id = appwrite_tablesdb.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.appwrite_tablesdb.test", "name", "Analytics"),
					resource.TestCheckResourceAttr("data.appwrite_tablesdb.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("data.appwrite_tablesdb.test", "created_at"),
				),
			},
		},
	})
}
