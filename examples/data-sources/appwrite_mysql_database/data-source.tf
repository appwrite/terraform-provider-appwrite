data "appwrite_mysql_database" "main" {
  id = "main"
}

output "database_host" {
  value = data.appwrite_mysql_database.main.hostname
}
