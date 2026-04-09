# Unique index
resource "appwrite_index" "email_unique" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_table.users.id
  key         = "email_unique"
  type        = "unique"
  columns     = [appwrite_column.email.key]
}

# Key index with sort order
resource "appwrite_index" "name_index" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_table.users.id
  key         = "name_index"
  type        = "key"
  columns     = [appwrite_column.name.key]
  orders      = ["asc"]
}
