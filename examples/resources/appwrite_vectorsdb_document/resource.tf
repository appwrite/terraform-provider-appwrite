resource "appwrite_vectorsdb" "main" {
  name = "embeddings"
}

resource "appwrite_vectorsdb_collection" "articles" {
  database_id = appwrite_vectorsdb.main.id
  name        = "Article embeddings"
  dimension   = 4
}

# The embedding must carry exactly `dimension` values, four in this toy example.
# Real embeddings come from a model, so they are usually written by the
# application rather than pinned in configuration.
resource "appwrite_vectorsdb_document" "seed" {
  database_id   = appwrite_vectorsdb.main.id
  collection_id = appwrite_vectorsdb_collection.articles.id
  id            = "seed"

  data = jsonencode({
    embedding = [0.1, 0.2, 0.3, 0.4]
    source_id = "article-1"
  })
}
