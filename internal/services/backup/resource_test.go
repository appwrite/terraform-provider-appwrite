package backup_test

import (
	"fmt"
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDatabaseBackupPolicyResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBackupPolicyConfig("daily-backup", "Daily Backup", 7, "0 2 * * *"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_database_backup_policy.test", "id", "daily-backup"),
					resource.TestCheckResourceAttr("appwrite_database_backup_policy.test", "name", "Daily Backup"),
					resource.TestCheckResourceAttr("appwrite_database_backup_policy.test", "retention", "7"),
					resource.TestCheckResourceAttr("appwrite_database_backup_policy.test", "schedule", "0 2 * * *"),
					resource.TestCheckResourceAttr("appwrite_database_backup_policy.test", "enabled", "true"),
					resource.TestCheckResourceAttr("appwrite_database_backup_policy.test", "services.#", "1"),
					resource.TestCheckResourceAttrSet("appwrite_database_backup_policy.test", "created_at"),
				),
			},
			{
				ResourceName:      "appwrite_database_backup_policy.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccBackupPolicyConfig("daily-backup", "Nightly Backup", 14, "0 3 * * *"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_database_backup_policy.test", "name", "Nightly Backup"),
					resource.TestCheckResourceAttr("appwrite_database_backup_policy.test", "retention", "14"),
					resource.TestCheckResourceAttr("appwrite_database_backup_policy.test", "schedule", "0 3 * * *"),
				),
			},
		},
	})
}

func testAccBackupPolicyConfig(id, name string, retention int, schedule string) string {
	return fmt.Sprintf(`
resource "appwrite_database_backup_policy" "test" {
  id        = %q
  name      = %q
  services  = ["databases"]
  retention = %d
  schedule  = %q
}
`, id, name, retention, schedule)
}
