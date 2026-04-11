resource "appwrite_tablesdb" "main" {
  name = "main"
}

resource "appwrite_tablesdb_table" "users" {
  database_id = appwrite_tablesdb.main.id
  name        = "users"
}

resource "appwrite_tablesdb_column" "email" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  key         = "email"
  type        = "email"
  required    = true
}

resource "appwrite_tablesdb_column" "name" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  key         = "name"
  type        = "varchar"
  size        = 255
  required    = true
}

resource "appwrite_tablesdb_index" "email_unique" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  key         = "email_unique"
  type        = "unique"
  columns     = [appwrite_tablesdb_column.email.key]
}

resource "appwrite_tablesdb_index" "name_index" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  key         = "name_index"
  type        = "key"
  columns     = [appwrite_tablesdb_column.name.key]
  orders      = ["asc"]
}
