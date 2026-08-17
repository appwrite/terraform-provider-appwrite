data "appwrite_postgresql_database_status" "main" {
  database_id = "main"
}

# These are measurements taken at refresh time, not configuration, so they
# change between runs on their own. Use them for outputs and checks rather than
# to drive resource arguments.
output "healthy" {
  value = data.appwrite_postgresql_database_status.main.health
}

output "connection_headroom" {
  value = format(
    "%d of %d connections in use",
    data.appwrite_postgresql_database_status.main.connections_current,
    data.appwrite_postgresql_database_status.main.connections_max,
  )
}

# Replication is only worth asserting on when the reading was actually taken:
# sync_state_confirmed = false means no measurement, not unhealthy replication.
output "lagging_replicas" {
  value = [
    for r in data.appwrite_postgresql_database_status.main.replicas :
    r.index if r.role == "replica" && r.lag_seconds > 30
  ]
}

check "database_is_healthy" {
  assert {
    condition     = data.appwrite_postgresql_database_status.main.ready
    error_message = "The main PostgreSQL database is not accepting connections."
  }
}
