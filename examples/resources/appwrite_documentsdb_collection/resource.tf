resource "appwrite_documentsdb" "main" {
  name = "main"
}

resource "appwrite_documentsdb_collection" "articles" {
  database_id = appwrite_documentsdb.main.id
  id          = "articles"
  name        = "Articles"

  # Collection-level permissions. Enable document_security as well to have
  # per-document permissions enforced on top of these.
  permissions       = ["read(\"any\")", "create(\"users\")"]
  document_security = true
}
