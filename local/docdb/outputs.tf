data "appwrite_documentsdb" "main" {
  id = appwrite_documentsdb.main.id
}

data "appwrite_vectorsdb" "main" {
  id = appwrite_vectorsdb.main.id
}

output "documentsdb" {
  value = {
    id      = data.appwrite_documentsdb.main.id
    name    = data.appwrite_documentsdb.main.name
    type    = data.appwrite_documentsdb.main.type
    enabled = data.appwrite_documentsdb.main.enabled
  }
}

output "vectorsdb" {
  value = {
    id      = data.appwrite_vectorsdb.main.id
    name    = data.appwrite_vectorsdb.main.name
    type    = data.appwrite_vectorsdb.main.type
    enabled = data.appwrite_vectorsdb.main.enabled
  }
}

output "vector_collection_dimension" {
  value = one(appwrite_vectorsdb_collection.embeddings[*].dimension)
}

output "index_status" {
  value = one(appwrite_documentsdb_index.by_slug[*].status)
}
