resource "appwrite_auth_team" "engineering" {
  name = "Engineering"
}

resource "appwrite_auth_user" "john" {
  name     = "John Doe"
  email    = "john@example.com"
  password = var.user_password
}

resource "appwrite_auth_user" "admin" {
  name     = "Admin"
  email    = "admin@example.com"
  password = var.admin_password
  labels   = ["admin", "staff"]
}
