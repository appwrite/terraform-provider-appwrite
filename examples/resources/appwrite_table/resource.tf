resource "appwrite_table" "users" {
  database_id = appwrite_database.main.id
  id          = "users"
  name        = "Users"
}

resource "appwrite_table" "posts" {
  database_id  = appwrite_database.main.id
  id           = "posts"
  name         = "Posts"
  row_security = true
}
