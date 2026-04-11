resource "appwrite_auth_team" "engineering" {
  name = "engineering"
}

resource "appwrite_auth_team" "marketing" {
  name  = "marketing"
  roles = ["owner", "editor"]
}
