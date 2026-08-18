resource "appwrite_mysql_database" "main" {
  name          = "main"
  specification = "s-2vcpu-4gb"
}

resource "appwrite_mysql_pooler" "main" {
  database_id       = appwrite_mysql_database.main.id
  mode              = "transaction"
  max_connections   = 200
  default_pool_size = 25
}
