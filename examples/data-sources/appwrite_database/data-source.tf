data "appwrite_database" "main" {
  id = "main"
}

output "database_name" {
  value = data.appwrite_database.main.name
}
