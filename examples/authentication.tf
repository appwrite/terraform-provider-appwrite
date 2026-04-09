resource "appwrite_auth_user" "john" {
  id       = "john"
  name     = "John Doe"
  email    = "john@example.com"
  password = "securepassword123"
  labels   = ["admin"]
}

resource "appwrite_auth_team" "engineering" {
  id   = "engineering"
  name = "Engineering"
}

resource "appwrite_auth_team" "marketing" {
  id    = "marketing"
  name  = "Marketing"
  roles = ["owner", "editor"]
}
