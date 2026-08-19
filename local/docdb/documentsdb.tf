# DocumentsDB: schemaless JSON documents.
#
# This deployment configures no shared pool, so both databases are created on
# dedicated infrastructure and are billable while they exist. Destroy when done.

resource "appwrite_documentsdb" "main" {
  id            = "${var.prefix}-docs"
  name          = "Sandbox DocumentsDB"
  specification = local.documentsdb_specification
}

resource "appwrite_documentsdb_collection" "articles" {
  count = var.manage_collections ? 1 : 0

  database_id       = appwrite_documentsdb.main.id
  id                = "articles"
  name              = "Articles"
  permissions       = ["read(\"any\")"]
  document_security = true

  # Attributes are create-only: there is no route to add one afterwards, so
  # anything to be indexed has to be declared here.
  attributes = jsonencode([
    {
      key      = "slug"
      type     = "string"
      size     = 255
      required = true
    },
    {
      key      = "title"
      type     = "string"
      size     = 512
      required = false
    },
    {
      key      = "views"
      type     = "integer"
      required = false
    },
  ])
}

resource "appwrite_documentsdb_index" "by_slug" {
  count = var.manage_collections ? 1 : 0

  database_id   = appwrite_documentsdb.main.id
  collection_id = appwrite_documentsdb_collection.articles[0].id
  key           = "by_slug"
  type          = "key"
  attributes    = ["slug"]
}

resource "appwrite_documentsdb_document" "seed" {
  count = var.manage_collections ? 1 : 0

  database_id   = appwrite_documentsdb.main.id
  collection_id = appwrite_documentsdb_collection.articles[0].id
  id            = "seed"

  data = jsonencode({
    slug  = "hello-world"
    title = "Hello, world"
    views = 1
  })
}
