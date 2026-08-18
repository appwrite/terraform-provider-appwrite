data "appwrite_mongo_database" "main" {
  id = "main"
}

output "database_host" {
  value = data.appwrite_mongo_database.main.hostname
}
