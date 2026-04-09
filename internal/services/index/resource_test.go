package index_test

import (
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccIndexResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_tablesdb" "test" {
  id   = "shop"
  name = "Shop"
}

resource "appwrite_tablesdb_table" "test" {
  database_id = appwrite_tablesdb.test.id
  id          = "orders"
  name        = "Orders"
}

resource "appwrite_tablesdb_column" "test" {
  database_id = appwrite_tablesdb.test.id
  table_id    = appwrite_tablesdb_table.test.id
  key         = "customer_name"
  type        = "varchar"
  size        = 255
  required    = true
}

resource "appwrite_tablesdb_index" "test" {
  database_id = appwrite_tablesdb.test.id
  table_id    = appwrite_tablesdb_table.test.id
  key         = "customer_name_index"
  type        = "key"
  columns     = [appwrite_tablesdb_column.test.key]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_tablesdb_index.test", "key", "customer_name_index"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_index.test", "type", "key"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_index.test", "columns.#", "1"),
					resource.TestCheckResourceAttr("appwrite_tablesdb_index.test", "columns.0", "customer_name"),
					resource.TestCheckResourceAttrSet("appwrite_tablesdb_index.test", "created_at"),
				),
			},
			{
				ResourceName:      "appwrite_tablesdb_index.test",
				ImportState:       true,
				ImportStateId:     "shop/orders/customer_name_index",
				ImportStateVerify: true,
			},
		},
	})
}
