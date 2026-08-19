resource "appwrite_documentsdb" "main" {
  name = "main"
}

# An index can only be built on a declared attribute, and attributes are
# create-only, so they are declared alongside the collection.
resource "appwrite_documentsdb_collection" "articles" {
  database_id = appwrite_documentsdb.main.id
  name        = "Articles"

  attributes = jsonencode([
    { key = "slug", type = "string", size = 255, required = true },
    { key = "published_at", type = "datetime", required = false },
  ])
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
