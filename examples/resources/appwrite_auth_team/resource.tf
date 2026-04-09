resource "appwrite_auth_user" "john" {
  name     = "John Doe"
  email    = "john@example.com"
  password = var.user_password
}

resource "appwrite_auth_team" "engineering" {
  name = "Engineering"
}

resource "appwrite_auth_team" "marketing" {
  name  = "Marketing"
  roles = ["owner", "editor"]
}
