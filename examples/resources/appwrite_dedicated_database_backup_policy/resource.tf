resource "appwrite_dedicated_database_backup_policy" "daily" {
  engine      = appwrite_dedicated_database.postgres.engine
  database_id = appwrite_dedicated_database.postgres.id
  name        = "daily"
  schedule    = "0 3 * * *"
  retention   = 30
  type        = "full"
}
