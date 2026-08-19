# A DocumentsDB database holds collections of schemaless JSON documents.
resource "appwrite_documentsdb" "main" {
  name = "main"
}

# Placing the database on dedicated infrastructure is opt-in and billed
# separately -- though a deployment with no shared pool requires it. Read the
# slug from this product's own catalogue: each product publishes its own, so a
# DocumentsDB database must be sized from the DocumentsDB specifications.
data "appwrite_documentsdb_specifications" "available" {}

resource "appwrite_documentsdb" "production" {
  name          = "production"
  specification = one([for s in data.appwrite_documentsdb_specifications.available.specifications : s.slug if s.enabled])
  replicas      = 1
  sync_mode     = "sync"
}
