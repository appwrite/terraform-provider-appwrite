package table_test

import (
	"fmt"
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTableResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTableConfig("ecommerce", "products", "Products"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_tablesdb_table.test", "id", "products"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_table.test", "name", "Products"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_table.test", "database_id", "ecommerce"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_table.test", "enabled", "true"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_table.test", "row_security", "false"),
					resource.TestCheckResourceAttrSet("appwrite_tablesdb_table.test", "created_at"),
				),
			},
			{
				ResourceName:      "appwrite_tablesdb_table.test",
				ImportState:       true,
				ImportStateId:     "ecommerce/products",
				ImportStateVerify: true,
			},
			{
				Config: testAccTableConfig("ecommerce", "products", "All Products"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_tablesdb_table.test", "name", "All Products"),
				),
			},
		},
	})
}

func testAccTableConfig(databaseId, tableId, name string) string {
	return fmt.Sprintf(`
resource "appwrite_tablesdb" "test" {
  id   = %q
  name = "E-Commerce"
}

resource "appwrite_tablesdb_table" "test" {
  database_id = appwrite_tablesdb.test.id
  id          = %q
  name        = %q
}
`, databaseId, tableId, name)
}
