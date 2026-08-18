# A DocumentsDB database holds collections of schemaless JSON documents.
resource "appwrite_documentsdb" "main" {
  name = "main"
}

# Placing the database on dedicated infrastructure is opt-in and billed
# separately. Read the slug from the API rather than hardcoding one that may not
# be available on your plan.
data "appwrite_postgresql_specifications" "available" {}

resource "appwrite_documentsdb" "production" {
  name          = "production"
  specification = one([for s in data.appwrite_postgresql_specifications.available.specifications : s.slug if s.enabled])
  replicas      = 1
  sync_mode     = "sync"
}
