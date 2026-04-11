resource "appwrite_auth_user" "john" {
  name     = "john doe"
  email    = "john@example.com"
  password = var.user_password
}

resource "appwrite_auth_user" "admin" {
  name     = "admin"
  email    = "admin@example.com"
  password = var.admin_password
  labels   = ["admin", "staff"]
}
