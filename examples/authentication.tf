resource "appwrite_auth_user" "john" {
  name     = "John Doe"
  email    = "john2@example.com"
  password = "securepassword123"
  labels   = ["admin"]
}

resource "appwrite_auth_team" "engineering" {
  name = "Engineering"
}

resource "appwrite_auth_team" "marketing" {
  name  = "Marketing"
  roles = ["owner", "editor"]
}
