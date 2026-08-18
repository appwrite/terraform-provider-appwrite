resource "appwrite_postgresql_database" "main" {
  name          = "main"
  specification = "s-1vcpu-1gb"
}

resource "appwrite_postgresql_extension" "postgis" {
  database_id = appwrite_postgresql_database.main.id
  name        = "postgis"
}

resource "appwrite_postgresql_extension" "pg_trgm" {
  database_id = appwrite_postgresql_database.main.id
  name        = "pg_trgm"
}

# The installable names come from the API, so a typo fails at apply rather than
# silently doing nothing.
data "appwrite_postgresql_extensions" "main" {
  database_id = appwrite_postgresql_database.main.id
}

output "available_extensions" {
  value = data.appwrite_postgresql_extensions.main.available
}
