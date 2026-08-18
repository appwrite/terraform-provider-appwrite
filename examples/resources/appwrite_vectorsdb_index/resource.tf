resource "appwrite_vectorsdb" "main" {
  name = "embeddings"
}

resource "appwrite_vectorsdb_collection" "articles" {
  database_id = appwrite_vectorsdb.main.id
  name        = "Article embeddings"
  dimension   = 1536
}

resource "appwrite_vectorsdb_index" "by_source" {
  database_id   = appwrite_vectorsdb.main.id
  collection_id = appwrite_vectorsdb_collection.articles.id
  key           = "by_source"
  type          = "key"
  attributes    = ["source_id"]
}
