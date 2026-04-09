package row_test

import (
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTablesDBRowResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRowConfig(`{"name": "Alice", "email": "alice@example.com"}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("appwrite_tablesdb_row.test", "id"),
					resource.TestCheckResourceAttrSet("appwrite_tablesdb_row.test", "created_at"),
				),
			},
			{
				ResourceName:      "appwrite_tablesdb_row.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccRowConfig(`{"name": "Alice Smith", "email": "alice@example.com"}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("appwrite_tablesdb_row.test", "updated_at"),
				),
			},
		},
	})
}

func testAccRowConfig(data string) string {
	return `
resource "appwrite_tablesdb" "test" {
  name = "row-test-db"
}

resource "appwrite_tablesdb_table" "test" {
  database_id = appwrite_tablesdb.test.id
  name        = "people"
}

resource "appwrite_tablesdb_column" "name" {
  database_id = appwrite_tablesdb.test.id
  table_id    = appwrite_tablesdb_table.test.id
  key         = "name"
  type        = "varchar"
  size        = 255
  required    = true
}

resource "appwrite_tablesdb_column" "email" {
  database_id = appwrite_tablesdb.test.id
  table_id    = appwrite_tablesdb_table.test.id
  key         = "email"
  type        = "email"
  required    = true
}

resource "appwrite_tablesdb_row" "test" {
  database_id = appwrite_tablesdb.test.id
  table_id    = appwrite_tablesdb_table.test.id
  data        = jsonencode(` + data + `)

  depends_on = [
    appwrite_tablesdb_column.name,
    appwrite_tablesdb_column.email,
  ]
}
`
}
