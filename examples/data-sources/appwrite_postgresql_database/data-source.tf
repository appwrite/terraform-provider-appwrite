data "appwrite_postgresql_database" "main" {
  id = "main"
}

# Hand the connection details to whatever needs them. The connection string
# embeds the password, so treat any output of it as a secret.
output "database_host" {
  value = data.appwrite_postgresql_database.main.hostname
}

output "database_connection_string" {
  value     = data.appwrite_postgresql_database.main.connection_string
  sensitive = true
}
