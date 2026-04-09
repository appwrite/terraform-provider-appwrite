resource "appwrite_table" "users" {
  database_id = appwrite_database.main.id
  id          = "users"
  name        = "users"
}

resource "appwrite_table" "posts" {
  database_id  = appwrite_database.main.id
  id           = "posts"
  name         = "posts"
  row_security = true
}
