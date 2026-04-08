resource "appwrite_database" "main" {
  id   = "main"
  name = "main"
}

resource "appwrite_database" "staging" {
  id      = "staging"
  name    = "staging"
  enabled = false
}
