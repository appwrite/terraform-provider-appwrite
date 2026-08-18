resource "appwrite_postgresql_database" "main" {
  name          = "main"
  specification = "s-1vcpu-1gb"
}

# A full snapshot every night, kept for a week.
resource "appwrite_postgresql_backup_policy" "nightly" {
  database_id = appwrite_postgresql_database.main.id
  name        = "nightly"
  schedule    = "0 3 * * *"
  retention   = 7
  type        = "full"
}

# Incremental backups every six hours narrow the recovery window without paying
# for a full snapshot each time.
resource "appwrite_postgresql_backup_policy" "incremental" {
  database_id = appwrite_postgresql_database.main.id
  name        = "six-hourly"
  schedule    = "0 */6 * * *"
  retention   = 3
  type        = "incremental"
}
