resource "appwrite_user" "john" {
  id       = "john"
  name     = "John Doe"
  email    = "john@example.com"
  password = var.user_password
}

resource "appwrite_user" "admin" {
  id       = "admin"
  name     = "Admin"
  email    = "admin@example.com"
  password = var.admin_password
  labels   = ["admin", "staff"]
}
