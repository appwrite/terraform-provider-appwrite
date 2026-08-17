package dedicated_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/appwrite/terraform-provider-appwrite/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var regexpIncompleteMaintenanceWindow = regexp.MustCompile(`Incomplete maintenance window`)

// specification returns the compute slug the acceptance tests provision with.
// It is configurable because the available slugs depend on the organization's
// billing plan.
func specification() string {
	if slug := os.Getenv("APPWRITE_DEDICATED_SPECIFICATION"); slug != "" {
		return slug
	}
	return "db-s-1vcpu-1gb"
}

func TestAccPostgresqlDatabaseResource_basic(t *testing.T) {
	databaseID := fmt.Sprintf("tf-pg-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.DedicatedPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPostgresqlDatabaseConfig(databaseID, "Terraform Postgres", 7),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_postgresql_database.test", "id", databaseID),
					resource.TestCheckResourceAttr("appwrite_postgresql_database.test", "name", "Terraform Postgres"),
					resource.TestCheckResourceAttr("appwrite_postgresql_database.test", "pitr_retention_days", "7"),
					resource.TestCheckResourceAttr("appwrite_postgresql_database.test", "engine", "postgresql"),
					// Create must not return until the database has left the
					// provisioning state, or dependent resources break.
					resource.TestCheckResourceAttr("appwrite_postgresql_database.test", "status", "ready"),
					resource.TestCheckResourceAttrSet("appwrite_postgresql_database.test", "hostname"),
					resource.TestCheckResourceAttrSet("appwrite_postgresql_database.test", "connection_port"),
					resource.TestCheckResourceAttrSet("appwrite_postgresql_database.test", "connection_string"),
					resource.TestCheckResourceAttrSet("appwrite_postgresql_database.test", "created_at"),
				),
			},
			{
				ResourceName:      "appwrite_postgresql_database.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccPostgresqlDatabaseConfig(databaseID, "Terraform Postgres v2", 14),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_postgresql_database.test", "name", "Terraform Postgres v2"),
					resource.TestCheckResourceAttr("appwrite_postgresql_database.test", "pitr_retention_days", "14"),
				),
			},
		},
	})
}

// The maintenance window lives behind its own route, so setting it at create
// time is the case most likely to regress.
func TestAccPostgresqlDatabaseResource_maintenanceWindow(t *testing.T) {
	databaseID := fmt.Sprintf("tf-pg-mw-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.DedicatedPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "appwrite_postgresql_database" "test" {
  id                          = %q
  name                        = "Terraform Postgres Maintenance"
  specification               = %q
  maintenance_window_day      = "sun"
  maintenance_window_hour_utc = 3
}
`, databaseID, specification()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_postgresql_database.test", "maintenance_window_day", "sun"),
					resource.TestCheckResourceAttr("appwrite_postgresql_database.test", "maintenance_window_hour_utc", "3"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "appwrite_postgresql_database" "test" {
  id                          = %q
  name                        = "Terraform Postgres Maintenance"
  specification               = %q
  maintenance_window_day      = "wed"
  maintenance_window_hour_utc = 11
}
`, databaseID, specification()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_postgresql_database.test", "maintenance_window_day", "wed"),
					resource.TestCheckResourceAttr("appwrite_postgresql_database.test", "maintenance_window_hour_utc", "11"),
				),
			},
		},
	})
}

// A day without an hour is rejected at apply time rather than sent as a partial
// pair the API would misread.
func TestAccPostgresqlDatabaseResource_incompleteMaintenanceWindow(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.DedicatedPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "appwrite_postgresql_database" "test" {
  id                     = "tf-pg-invalid"
  name                   = "Terraform Postgres Invalid"
  specification          = %q
  maintenance_window_day = "sun"
}
`, specification()),
				ExpectError: regexpIncompleteMaintenanceWindow,
			},
		},
	})
}

func TestAccMysqlDatabaseResource_basic(t *testing.T) {
	databaseID := fmt.Sprintf("tf-my-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.DedicatedPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "appwrite_mysql_database" "test" {
  id            = %q
  name          = "Terraform MySQL"
  specification = %q
}
`, databaseID, specification()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_mysql_database.test", "id", databaseID),
					resource.TestCheckResourceAttr("appwrite_mysql_database.test", "status", "ready"),
					resource.TestCheckResourceAttrSet("appwrite_mysql_database.test", "hostname"),
				),
			},
		},
	})
}

func TestAccMongoDatabaseResource_basic(t *testing.T) {
	databaseID := fmt.Sprintf("tf-mongo-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.DedicatedPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "appwrite_mongo_database" "test" {
  id            = %q
  name          = "Terraform Mongo"
  specification = %q
}
`, databaseID, specification()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("appwrite_mongo_database.test", "id", databaseID),
					resource.TestCheckResourceAttr("appwrite_mongo_database.test", "status", "ready"),
					resource.TestCheckResourceAttrSet("appwrite_mongo_database.test", "hostname"),
				),
			},
		},
	})
}

func testAccPostgresqlDatabaseConfig(databaseID, name string, pitrRetentionDays int) string {
	return fmt.Sprintf(`
resource "appwrite_postgresql_database" "test" {
  id                  = %q
  name                = %q
  specification       = %q
  pitr                = true
  pitr_retention_days = %d
}
`, databaseID, name, specification(), pitrRetentionDays)
}
