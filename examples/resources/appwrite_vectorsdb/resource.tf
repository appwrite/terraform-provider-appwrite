# A VectorsDB database holds collections of embeddings searched by similarity.
resource "appwrite_vectorsdb" "main" {
  name = "embeddings"
}
