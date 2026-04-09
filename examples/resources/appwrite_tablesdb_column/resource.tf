# Varchar column with max length
resource "appwrite_column" "name" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_table.users.id
  key         = "name"
  type        = "varchar"
  size        = 255
  required    = true
}

# Email column
resource "appwrite_column" "email" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_table.users.id
  key         = "email"
  type        = "email"
  required    = true
}

# Integer column with min/max
resource "appwrite_column" "age" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_table.users.id
  key         = "age"
  type        = "integer"
  min         = 0
  max         = 150
}

# Boolean column with default
resource "appwrite_column" "active" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_table.users.id
  key         = "active"
  type        = "boolean"
  default     = "true"
}

# Float column with min/max
resource "appwrite_column" "score" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_table.users.id
  key         = "score"
  type        = "float"
  float_min   = 0.0
  float_max   = 100.0
}

# Enum column with allowed values
resource "appwrite_column" "role" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_table.users.id
  key         = "role"
  type        = "enum"
  elements    = ["admin", "editor", "viewer"]
  default     = "viewer"
}

# Datetime column
resource "appwrite_column" "joined_at" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_table.users.id
  key         = "joined_at"
  type        = "datetime"
}

# Array column
resource "appwrite_column" "tags" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_table.users.id
  key         = "tags"
  type        = "varchar"
  size        = 64
  array       = true
}

# Geographic point column
resource "appwrite_column" "location" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_table.users.id
  key         = "location"
  type        = "point"
}

# Relationship column
resource "appwrite_column" "author" {
  database_id       = appwrite_tablesdb.main.id
  table_id          = appwrite_table.posts.id
  related_table_id  = appwrite_table.users.id
  type              = "relationship"
  relationship_type = "manyToOne"
  two_way           = true
  two_way_key       = "posts"
  on_delete         = "cascade"
}
