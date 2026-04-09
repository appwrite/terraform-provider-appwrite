resource "appwrite_database" "main" {
  id   = "main-00002"
  name = "main"
}

resource "appwrite_database_table" "users" {
  database_id = appwrite_database.main.id
  id          = "users"
  name        = "users"
}

resource "appwrite_database_column" "name" {
  database_id = appwrite_database.main.id
  table_id    = appwrite_database_table.users.id
  key         = "name"
  type        = "varchar"
  size        = 255
  required    = true
}

resource "appwrite_database_column" "email" {
  database_id = appwrite_database.main.id
  table_id    = appwrite_database_table.users.id
  key         = "email"
  type        = "email"
  required    = true
}

resource "appwrite_database_column" "age" {
  database_id = appwrite_database.main.id
  table_id    = appwrite_database_table.users.id
  key         = "age"
  type        = "integer"
  min         = 0
  max         = 150
}

resource "appwrite_database_column" "active" {
  database_id = appwrite_database.main.id
  table_id    = appwrite_database_table.users.id
  key         = "active"
  type        = "boolean"
  default     = "true"
}

resource "appwrite_database_column" "score" {
  database_id = appwrite_database.main.id
  table_id    = appwrite_database_table.users.id
  key         = "score"
  type        = "float"
  float_min    = 0.0
  float_max    = 100.0
  default     = "0"
}

resource "appwrite_database_column" "role" {
  database_id = appwrite_database.main.id
  table_id    = appwrite_database_table.users.id
  key         = "role"
  type        = "enum"
  elements    = ["admin", "editor", "viewer"]
  default     = "viewer"
}

resource "appwrite_database_column" "joined_at" {
  database_id = appwrite_database.main.id
  table_id    = appwrite_database_table.users.id
  key         = "joined_at"
  type        = "datetime"
}

resource "appwrite_database_column" "website" {
  database_id = appwrite_database.main.id
  table_id    = appwrite_database_table.users.id
  key         = "website"
  type        = "url"
}

resource "appwrite_database_column" "last_login_ip" {
  database_id = appwrite_database.main.id
  table_id    = appwrite_database_table.users.id
  key         = "last_login_ip"
  type        = "ip"
}

resource "appwrite_database_column" "bio" {
  database_id = appwrite_database.main.id
  table_id    = appwrite_database_table.users.id
  key         = "bio"
  type        = "text"
}

resource "appwrite_database_column" "notes" {
  database_id = appwrite_database.main.id
  table_id    = appwrite_database_table.users.id
  key         = "notes"
  type        = "longtext"
}

resource "appwrite_database_column" "nickname" {
  database_id = appwrite_database.main.id
  table_id    = appwrite_database_table.users.id
  key         = "nickname"
  type        = "varchar"
  size        = 64
}

resource "appwrite_database_column" "tags" {
  database_id = appwrite_database.main.id
  table_id    = appwrite_database_table.users.id
  key         = "tags"
  type        = "varchar"
  size        = 64
  array       = true
}

resource "appwrite_database_column" "location" {
  database_id = appwrite_database.main.id
  table_id    = appwrite_database_table.users.id
  key         = "location"
  type        = "point"
}

resource "appwrite_database_index" "email_unique" {
  database_id = appwrite_database.main.id
  table_id    = appwrite_database_table.users.id
  key         = "email_unique"
  type        = "unique"
  columns     = [appwrite_database_column.email.key]
}
