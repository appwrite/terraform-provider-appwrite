resource "appwrite_team" "engineering" {
  name = "Engineering"
}

resource "appwrite_team" "marketing" {
  name  = "Marketing"
  roles = ["owner", "editor"]
}
