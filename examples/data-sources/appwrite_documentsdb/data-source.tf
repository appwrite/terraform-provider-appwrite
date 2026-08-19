data "appwrite_documentsdb" "main" {
  id = "main"
}

output "database_name" {
  value = data.appwrite_documentsdb.main.name
}

# Empty unless the database was placed on dedicated infrastructure.
output "backing_engine" {
  value = data.appwrite_documentsdb.main.engine
}
