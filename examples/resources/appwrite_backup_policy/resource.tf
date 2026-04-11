resource "appwrite_backup_policy" "daily" {
  name      = "daily database backup"
  services  = ["databases"]
  retention = 7
  schedule  = "0 2 * * *"
}

resource "appwrite_tablesdb" "main" {
  name = "main"
}

resource "appwrite_backup_policy" "production" {
  name        = "production database backup"
  services    = ["databases"]
  resource_id = appwrite_tablesdb.main.id
  retention   = 14
  schedule    = "0 */6 * * *"
}
