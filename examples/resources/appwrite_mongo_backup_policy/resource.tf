resource "appwrite_mongo_database" "main" {
  name          = "main"
  specification = "db-s-1vcpu-1gb"
}

resource "appwrite_mongo_backup_policy" "nightly" {
  database_id = appwrite_mongo_database.main.id
  name        = "nightly"
  schedule    = "0 3 * * *"
  retention   = 7
  type        = "full"
}
