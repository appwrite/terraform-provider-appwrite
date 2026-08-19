data "appwrite_vectorsdb" "main" {
  id = "embeddings"
}

output "database_name" {
  value = data.appwrite_vectorsdb.main.name
}
