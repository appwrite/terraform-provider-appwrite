resource "appwrite_tablesdb" "main" {
  name = "main"
}

resource "appwrite_tablesdb_table" "users" {
  database_id = appwrite_tablesdb.main.id
  name        = "users"
}

resource "appwrite_tablesdb_column" "name" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  key         = "name"
  type        = "varchar"
  size        = 255
  required    = true
}

resource "appwrite_tablesdb_column" "email" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  key         = "email"
  type        = "email"
  required    = true
}

resource "appwrite_tablesdb_column" "age" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  key         = "age"
  type        = "integer"
  min         = 0
  max         = 150
}

resource "appwrite_tablesdb_column" "active" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  key         = "active"
  type        = "boolean"
  default     = "true"
}

resource "appwrite_tablesdb_column" "score" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  key         = "score"
  type        = "float"
  float_min   = 0.0
  float_max   = 100.0
  default     = "0"
}

resource "appwrite_tablesdb_column" "role" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  key         = "role"
  type        = "enum"
  elements    = ["admin", "editor", "viewer"]
  default     = "viewer"
}

resource "appwrite_tablesdb_column" "joined_at" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  key         = "joined_at"
  type        = "datetime"
}

resource "appwrite_tablesdb_column" "website" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  key         = "website"
  type        = "url"
}

resource "appwrite_tablesdb_column" "last_login_ip" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  key         = "last_login_ip"
  type        = "ip"
}

resource "appwrite_tablesdb_column" "bio" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  key         = "bio"
  type        = "text"
}

resource "appwrite_tablesdb_column" "notes" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  key         = "notes"
  type        = "longtext"
}

resource "appwrite_tablesdb_column" "nickname" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  key         = "nickname"
  type        = "varchar"
  size        = 64
}

resource "appwrite_tablesdb_column" "tags" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  key         = "tags"
  type        = "varchar"
  size        = 64
  array       = true
}

resource "appwrite_tablesdb_column" "location" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  key         = "location"
  type        = "point"
}

resource "appwrite_tablesdb_index" "email_unique" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  key         = "email_unique"
  type        = "unique"
  columns     = [appwrite_tablesdb_column.email.key]
}

resource "appwrite_tablesdb_row" "alice" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  data = jsonencode({
    name   = "Alice"
    email  = "alice@example.com"
    age    = 30
    active = true
    role   = "admin"
  })

  depends_on = [
    appwrite_tablesdb_column.name,
    appwrite_tablesdb_column.email,
    appwrite_tablesdb_column.age,
    appwrite_tablesdb_column.active,
    appwrite_tablesdb_column.role,
  ]
}

resource "appwrite_tablesdb_row" "bob" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  data = jsonencode({
    name   = "Bob"
    email  = "bob@example.com"
    age    = 25
    active = true
    role   = "editor"
  })

  depends_on = [
    appwrite_tablesdb_column.name,
    appwrite_tablesdb_column.email,
    appwrite_tablesdb_column.age,
    appwrite_tablesdb_column.active,
    appwrite_tablesdb_column.role,
  ]
}

resource "appwrite_tablesdb_row" "charlie" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  data = jsonencode({
    name   = "Charlie"
    email  = "charlie@example.com"
    age    = 35
    active = true
    role   = "viewer"
  })

  depends_on = [
    appwrite_tablesdb_column.name,
    appwrite_tablesdb_column.email,
    appwrite_tablesdb_column.age,
    appwrite_tablesdb_column.active,
    appwrite_tablesdb_column.role,
  ]
}

resource "appwrite_tablesdb_row" "diana" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  data = jsonencode({
    name   = "Diana"
    email  = "diana@example.com"
    age    = 28
    active = true
    role   = "admin"
  })

  depends_on = [
    appwrite_tablesdb_column.name,
    appwrite_tablesdb_column.email,
    appwrite_tablesdb_column.age,
    appwrite_tablesdb_column.active,
    appwrite_tablesdb_column.role,
  ]
}

resource "appwrite_tablesdb_row" "ethan" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  data = jsonencode({
    name   = "Ethan"
    email  = "ethan@example.com"
    age    = 42
    active = false
    role   = "viewer"
  })

  depends_on = [
    appwrite_tablesdb_column.name,
    appwrite_tablesdb_column.email,
    appwrite_tablesdb_column.age,
    appwrite_tablesdb_column.active,
    appwrite_tablesdb_column.role,
  ]
}

resource "appwrite_tablesdb_row" "fiona" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  data = jsonencode({
    name   = "Fiona"
    email  = "fiona@example.com"
    age    = 31
    active = true
    role   = "editor"
  })

  depends_on = [
    appwrite_tablesdb_column.name,
    appwrite_tablesdb_column.email,
    appwrite_tablesdb_column.age,
    appwrite_tablesdb_column.active,
    appwrite_tablesdb_column.role,
  ]
}

resource "appwrite_tablesdb_row" "george" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  data = jsonencode({
    name   = "George"
    email  = "george@example.com"
    age    = 50
    active = true
    role   = "viewer"
  })

  depends_on = [
    appwrite_tablesdb_column.name,
    appwrite_tablesdb_column.email,
    appwrite_tablesdb_column.age,
    appwrite_tablesdb_column.active,
    appwrite_tablesdb_column.role,
  ]
}

resource "appwrite_tablesdb_row" "hannah" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  data = jsonencode({
    name   = "Hannah"
    email  = "hannah@example.com"
    age    = 22
    active = true
    role   = "editor"
  })

  depends_on = [
    appwrite_tablesdb_column.name,
    appwrite_tablesdb_column.email,
    appwrite_tablesdb_column.age,
    appwrite_tablesdb_column.active,
    appwrite_tablesdb_column.role,
  ]
}

resource "appwrite_tablesdb_row" "ivan" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  data = jsonencode({
    name   = "Ivan"
    email  = "ivan@example.com"
    age    = 38
    active = false
    role   = "viewer"
  })

  depends_on = [
    appwrite_tablesdb_column.name,
    appwrite_tablesdb_column.email,
    appwrite_tablesdb_column.age,
    appwrite_tablesdb_column.active,
    appwrite_tablesdb_column.role,
  ]
}

resource "appwrite_tablesdb_row" "julia" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  data = jsonencode({
    name   = "Julia"
    email  = "julia@example.com"
    age    = 27
    active = true
    role   = "admin"
  })

  depends_on = [
    appwrite_tablesdb_column.name,
    appwrite_tablesdb_column.email,
    appwrite_tablesdb_column.age,
    appwrite_tablesdb_column.active,
    appwrite_tablesdb_column.role,
  ]
}

resource "appwrite_tablesdb_row" "kevin" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  data = jsonencode({
    name   = "Kevin"
    email  = "kevin@example.com"
    age    = 33
    active = true
    role   = "viewer"
  })

  depends_on = [
    appwrite_tablesdb_column.name,
    appwrite_tablesdb_column.email,
    appwrite_tablesdb_column.age,
    appwrite_tablesdb_column.active,
    appwrite_tablesdb_column.role,
  ]
}

resource "appwrite_tablesdb_row" "laura" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  data = jsonencode({
    name   = "Laura"
    email  = "laura@example.com"
    age    = 45
    active = true
    role   = "editor"
  })

  depends_on = [
    appwrite_tablesdb_column.name,
    appwrite_tablesdb_column.email,
    appwrite_tablesdb_column.age,
    appwrite_tablesdb_column.active,
    appwrite_tablesdb_column.role,
  ]
}

resource "appwrite_tablesdb_row" "mike" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  data = jsonencode({
    name   = "Mike"
    email  = "mike@example.com"
    age    = 29
    active = false
    role   = "viewer"
  })

  depends_on = [
    appwrite_tablesdb_column.name,
    appwrite_tablesdb_column.email,
    appwrite_tablesdb_column.age,
    appwrite_tablesdb_column.active,
    appwrite_tablesdb_column.role,
  ]
}

resource "appwrite_tablesdb_row" "nina" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  data = jsonencode({
    name   = "Nina"
    email  = "nina@example.com"
    age    = 36
    active = true
    role   = "editor"
  })

  depends_on = [
    appwrite_tablesdb_column.name,
    appwrite_tablesdb_column.email,
    appwrite_tablesdb_column.age,
    appwrite_tablesdb_column.active,
    appwrite_tablesdb_column.role,
  ]
}

resource "appwrite_tablesdb_row" "oscar" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  data = jsonencode({
    name   = "Oscar"
    email  = "oscar@example.com"
    age    = 41
    active = true
    role   = "viewer"
  })

  depends_on = [
    appwrite_tablesdb_column.name,
    appwrite_tablesdb_column.email,
    appwrite_tablesdb_column.age,
    appwrite_tablesdb_column.active,
    appwrite_tablesdb_column.role,
  ]
}

resource "appwrite_tablesdb_row" "paula" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  data = jsonencode({
    name   = "Paula"
    email  = "paula@example.com"
    age    = 24
    active = true
    role   = "admin"
  })

  depends_on = [
    appwrite_tablesdb_column.name,
    appwrite_tablesdb_column.email,
    appwrite_tablesdb_column.age,
    appwrite_tablesdb_column.active,
    appwrite_tablesdb_column.role,
  ]
}

resource "appwrite_tablesdb_row" "quinn" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  data = jsonencode({
    name   = "Quinn"
    email  = "quinn@example.com"
    age    = 19
    active = true
    role   = "viewer"
  })

  depends_on = [
    appwrite_tablesdb_column.name,
    appwrite_tablesdb_column.email,
    appwrite_tablesdb_column.age,
    appwrite_tablesdb_column.active,
    appwrite_tablesdb_column.role,
  ]
}

resource "appwrite_tablesdb_row" "rachel" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  data = jsonencode({
    name   = "Rachel"
    email  = "rachel@example.com"
    age    = 34
    active = false
    role   = "editor"
  })

  depends_on = [
    appwrite_tablesdb_column.name,
    appwrite_tablesdb_column.email,
    appwrite_tablesdb_column.age,
    appwrite_tablesdb_column.active,
    appwrite_tablesdb_column.role,
  ]
}

resource "appwrite_tablesdb_row" "steve" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  data = jsonencode({
    name   = "Steve"
    email  = "steve@example.com"
    age    = 48
    active = true
    role   = "viewer"
  })

  depends_on = [
    appwrite_tablesdb_column.name,
    appwrite_tablesdb_column.email,
    appwrite_tablesdb_column.age,
    appwrite_tablesdb_column.active,
    appwrite_tablesdb_column.role,
  ]
}

resource "appwrite_tablesdb_row" "tina" {
  database_id = appwrite_tablesdb.main.id
  table_id    = appwrite_tablesdb_table.users.id
  data = jsonencode({
    name   = "Tina"
    email  = "tina@example.com"
    age    = 26
    active = true
    role   = "editor"
  })

  depends_on = [
    appwrite_tablesdb_column.name,
    appwrite_tablesdb_column.email,
    appwrite_tablesdb_column.age,
    appwrite_tablesdb_column.active,
    appwrite_tablesdb_column.role,
  ]
}
