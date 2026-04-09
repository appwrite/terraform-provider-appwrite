resource "appwrite_auth_team" "engineering" {
  name = "Engineering"
}

resource "appwrite_auth_team" "marketing" {
  name  = "Marketing"
  roles = ["owner", "editor"]
}
