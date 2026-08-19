resource "appwrite_vectorsdb" "main" {
  name = "embeddings"
}

# dimension is required and must match the model producing the embeddings.
# text-embedding-3-small emits 1536 values; changing it later re-indexes the
# collection, so pick it deliberately.
resource "appwrite_vectorsdb_collection" "articles" {
  database_id = appwrite_vectorsdb.main.id
  id          = "article-embeddings"
  name        = "Article embeddings"
  dimension   = 1536

  permissions = ["read(\"any\")"]
}
