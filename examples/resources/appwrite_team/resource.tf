resource "appwrite_team" "engineering" {
  id   = "engineering"
  name = "Engineering"
}

resource "appwrite_team" "marketing" {
  id    = "marketing"
  name  = "Marketing"
  roles = ["owner", "editor"]
}
