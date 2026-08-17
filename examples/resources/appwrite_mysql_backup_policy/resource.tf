resource "appwrite_mysql_database" "main" {
  name          = "main"
  specification = "db-s-1vcpu-1gb"
}

resource "appwrite_mysql_backup_policy" "nightly" {
  database_id = appwrite_mysql_database.main.id
  name        = "nightly"
  schedule    = "0 3 * * *"
  retention   = 7
  type        = "full"
}
