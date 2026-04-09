# Daily database backup with 7 day retention
resource "appwrite_database_backup_policy" "daily" {
  id        = "daily-db-backup"
  name      = "Daily Database Backup"
  services  = ["databases"]
  retention = 7
  schedule  = "0 2 * * *"
}

# Backup a specific database
resource "appwrite_database_backup_policy" "production" {
  id          = "production-backup"
  name        = "Production Database Backup"
  services    = ["databases"]
  resource_id = appwrite_database.main.id
  retention   = 14
  schedule    = "0 */6 * * *"
}
