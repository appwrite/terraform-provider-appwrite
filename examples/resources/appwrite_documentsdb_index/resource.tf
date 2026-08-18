resource "appwrite_documentsdb" "main" {
  name = "main"
}

resource "appwrite_documentsdb_collection" "articles" {
  database_id = appwrite_documentsdb.main.id
  name        = "Articles"
}

# Indexes have no update route, so changing any argument replaces the index.
# Creation is asynchronous; Terraform waits for it to become available.
resource "appwrite_documentsdb_index" "by_slug" {
  database_id   = appwrite_documentsdb.main.id
  collection_id = appwrite_documentsdb_collection.articles.id
  key           = "by_slug"
  type          = "unique"
  attributes    = ["slug"]
}

resource "appwrite_documentsdb_index" "by_published" {
  database_id   = appwrite_documentsdb.main.id
  collection_id = appwrite_documentsdb_collection.articles.id
  key           = "by_published"
  type          = "key"
  attributes    = ["published_at"]
  orders        = ["DESC"]
}
