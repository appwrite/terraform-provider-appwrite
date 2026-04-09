package database_test

import (
	"fmt"
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDatabaseResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDatabaseConfig("inventory", "Inventory"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_tablesdb.test", "id", "inventory"),
					resource.TestCheckResourceAttr("appwrite_tablesdb.test", "name", "Inventory"),
					resource.TestCheckResourceAttr("appwrite_tablesdb.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("appwrite_tablesdb.test", "created_at"),
					resource.TestCheckResourceAttrSet("appwrite_tablesdb.test", "updated_at"),
				),
			},
			{
				ResourceName:      "appwrite_tablesdb.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccDatabaseConfig("inventory", "Inventory v2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_tablesdb.test", "name", "Inventory v2"),
				),
			},
		},
	})
}

func TestAccDatabaseResource_disabled(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "appwrite_tablesdb" "test" {
  id      = "staging"
  name    = "Staging"
  enabled = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_tablesdb.test", "enabled", "false"),
				),
			},
		},
	})
}

func testAccDatabaseConfig(id, name string) string {
	return fmt.Sprintf(`
resource "appwrite_tablesdb" "test" {
  id   = %q
  name = %q
}
`, id, name)
}
