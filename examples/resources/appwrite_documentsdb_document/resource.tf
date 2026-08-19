resource "appwrite_documentsdb" "main" {
  name = "main"
}

resource "appwrite_documentsdb_collection" "settings" {
  database_id = appwrite_documentsdb.main.id
  name        = "Settings"
}

# Documents managed here suit seed and reference data. Records an application
# writes at runtime should not be managed by Terraform, or every apply will
# fight the application.
resource "appwrite_documentsdb_document" "defaults" {
  database_id   = appwrite_documentsdb.main.id
  collection_id = appwrite_documentsdb_collection.settings.id
  id            = "defaults"

  data = jsonencode({
    theme       = "dark"
    locale      = "en-GB"
    max_uploads = 25
  })
}

# Only the keys present in `data` are tracked, so fields added by other clients
# do not show up as drift.
