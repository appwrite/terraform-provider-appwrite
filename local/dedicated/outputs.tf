# The specification each database landed on, and what it cost to pick it.
output "chosen_specifications" {
  description = "The compute slug used per engine."
  value = {
    postgresql = local.postgresql_specification
    mysql      = local.mysql_specification
    mongo      = local.mongo_specification
  }
}

output "available_specifications" {
  description = "Every slug the billing plan allows, per engine. Useful when pinning one."
  value = {
    postgresql = [for s in data.appwrite_postgresql_specifications.available.specifications : s.slug if s.enabled]
    mysql      = [for s in data.appwrite_mysql_specifications.available.specifications : s.slug if s.enabled]
    mongo      = [for s in data.appwrite_mongo_specifications.available.specifications : s.slug if s.enabled]
  }
}

output "endpoints" {
  description = "Where each database listens."
  value = {
    postgresql = "${appwrite_postgresql_database.main.hostname}:${appwrite_postgresql_database.main.connection_port}"
    mysql      = "${appwrite_mysql_database.main.hostname}:${appwrite_mysql_database.main.connection_port}"
    mongo      = "${appwrite_mongo_database.main.hostname}:${appwrite_mongo_database.main.connection_port}"
  }
}

output "statuses" {
  description = "The status each database settled into after the apply."
  value = {
    postgresql = appwrite_postgresql_database.main.status
    mysql      = appwrite_mysql_database.main.status
    mongo      = appwrite_mongo_database.main.status
  }
}

output "connection_strings" {
  description = "Full connection URIs, including credentials."
  sensitive   = true
  value = {
    postgresql = appwrite_postgresql_database.main.connection_string
    mysql      = appwrite_mysql_database.main.connection_string
    mongo      = appwrite_mongo_database.main.connection_string
  }
}

output "installed_extensions" {
  description = "Extensions installed on the PostgreSQL database."
  value       = data.appwrite_postgresql_extensions.main.installed
}

output "health" {
  description = "Live health reported by each database. Measured at refresh time, so it moves on its own."
  value = {
    postgresql = data.appwrite_postgresql_database_status.main.health
    mysql      = data.appwrite_mysql_database_status.main.health
    mongo      = data.appwrite_mongo_database_status.main.health
  }
}
