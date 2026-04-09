resource "appwrite_tablesdb" "main" {
  name = "main"
}

resource "appwrite_tablesdb_table" "users" {
  database_id = appwrite_tablesdb.main.id
  name        = "users"
}

resource "appwrite_tablesdb_table" "posts" {
  database_id  = appwrite_tablesdb.main.id
  name         = "posts"
  row_security = true
}
