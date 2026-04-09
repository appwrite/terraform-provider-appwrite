data "appwrite_database" "main" {
  id = appwrite_database.main.id
}

data "appwrite_health" "status" {}
