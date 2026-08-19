# VectorsDB: embeddings searched by similarity. A collection fixes the
# embedding dimension, and every document must match it exactly.

resource "appwrite_vectorsdb" "main" {
  id            = "${var.prefix}-vectors"
  name          = "Sandbox VectorsDB"
  specification = local.vectorsdb_specification
}

resource "appwrite_vectorsdb_collection" "embeddings" {
  count = var.manage_collections ? 1 : 0

  database_id = appwrite_vectorsdb.main.id
  id          = "embeddings"
  name        = "Embeddings"
  dimension   = var.dimension
  permissions = ["read(\"any\")"]
}

resource "appwrite_vectorsdb_document" "seed" {
  count = var.manage_collections ? 1 : 0

  database_id   = appwrite_vectorsdb.main.id
  collection_id = appwrite_vectorsdb_collection.embeddings[0].id
  id            = "seed"

  data = jsonencode({
    embedding = [0.1, 0.2, 0.3, 0.4]
    source_id = "article-1"
  })
}
