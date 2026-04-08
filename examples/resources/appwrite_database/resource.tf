resource "appwrite_database" "main" {
  id   = "main"
  name = "Main"
}

resource "appwrite_database" "staging" {
  id      = "staging"
  name    = "Staging"
  enabled = false
}
