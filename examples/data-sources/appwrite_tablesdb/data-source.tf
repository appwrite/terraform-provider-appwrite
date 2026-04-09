data "appwrite_tablesdb" "main" {
  id = "main"
}

output "database_name" {
  value = data.appwrite_tablesdb.main.name
}
