data "appwrite_mongo_database_status" "main" {
  database_id = "main"
}

# Measurements taken at refresh time, not configuration, so they change between
# runs on their own.
output "health" {
  value = data.appwrite_mongo_database_status.main.health
}

check "database_is_healthy" {
  assert {
    condition     = data.appwrite_mongo_database_status.main.ready
    error_message = "The main MongoDB database is not accepting connections."
  }
}
