resource "appwrite_table" "users" {
  database_id = appwrite_tablesdb.main.id
  name        = "users"
}

resource "appwrite_table" "posts" {
  database_id  = appwrite_tablesdb.main.id
  name         = "posts"
  row_security = true
}
