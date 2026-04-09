resource "appwrite_tablesdb" "main" {
  name = "main"
}

resource "appwrite_tablesdb" "staging" {
  name    = "staging"
  enabled = false
}
