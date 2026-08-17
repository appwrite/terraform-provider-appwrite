resource "appwrite_postgresql_database" "main" {
  name          = "main"
  specification = "db-s-2vcpu-4gb"
  replicas      = 1
}

# Transaction pooling suits short-lived queries from serverless functions, where
# holding a connection for a whole session would exhaust the pool.
#
# max_connections is not settable on PostgreSQL: the pooler has no client cap of
# its own and reports the database's network_max_connections instead.
resource "appwrite_postgresql_pooler" "main" {
  database_id       = appwrite_postgresql_database.main.id
  mode              = "transaction"
  default_pool_size = 25

  # Send SELECTs to the replica and keep writes on the primary.
  read_write_splitting = true
}
