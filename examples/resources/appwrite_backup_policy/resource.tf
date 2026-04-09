# Daily backup of all databases with 7 day retention
resource "appwrite_backup_policy" "daily" {
  name      = "Daily Database Backup"
  services  = ["databases"]
  retention = 7
  schedule  = "0 2 * * *"
}

# Backup a specific database
resource "appwrite_tablesdb" "main" {
  name = "main"
}

resource "appwrite_backup_policy" "production" {
  name        = "Production Database Backup"
  services    = ["databases"]
  resource_id = appwrite_tablesdb.main.id
  retention   = 14
  schedule    = "0 */6 * * *"
}
