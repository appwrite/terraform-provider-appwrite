resource "appwrite_documentsdb" "main" {
  name = "main"
}

resource "appwrite_documentsdb_collection" "articles" {
  database_id = appwrite_documentsdb.main.id
  id          = "articles"
  name        = "Articles"

  # Collection-level permissions. Enable document_security as well to have
  # per-document permissions enforced on top of these.
  permissions       = ["read(\"any\")", "create(\"users\")"]
  document_security = true

  # Attributes can only be declared when the collection is created -- there is
  # no route to add, change or remove one afterwards -- so changing this
  # replaces the collection. An index can only be built on a declared
  # attribute, so declare anything you intend to index.
  attributes = jsonencode([
    {
      key      = "slug"
      type     = "string"
      size     = 255
      required = true
    },
    {
      key      = "published_at"
      type     = "datetime"
      required = false
    },
  ])
}
