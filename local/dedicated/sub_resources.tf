# Everything that hangs off a dedicated database. These are cheap next to the
# databases themselves, so they are on by default.

# --- Backup policies: all three engines -------------------------------------

resource "appwrite_postgresql_backup_policy" "nightly" {
  database_id = appwrite_postgresql_database.main.id
  name        = "nightly"
  schedule    = "0 3 * * *"
  retention   = 7
  type        = "full"
}

resource "appwrite_mysql_backup_policy" "nightly" {
  database_id = appwrite_mysql_database.main.id
  name        = "nightly"
  schedule    = "0 4 * * *"
  retention   = 7
  type        = "full"
}

resource "appwrite_mongo_backup_policy" "nightly" {
  database_id = appwrite_mongo_database.main.id
  name        = "nightly"
  schedule    = "0 5 * * *"
  retention   = 7
  type        = "full"
}

# --- Poolers: PostgreSQL and MySQL only, MongoDB has none -------------------

resource "appwrite_postgresql_pooler" "main" {
  database_id       = appwrite_postgresql_database.main.id
  mode              = "transaction"
  default_pool_size = 25

  # max_connections is intentionally absent: it is read-only on PostgreSQL,
  # where the pooler has no client cap of its own. Setting it fails at plan time.
}

resource "appwrite_mysql_pooler" "main" {
  database_id       = appwrite_mysql_database.main.id
  mode              = "transaction"
  default_pool_size = 25
  max_connections   = 100
}

# --- Extensions: PostgreSQL only --------------------------------------------

resource "appwrite_postgresql_extension" "pg_trgm" {
  database_id = appwrite_postgresql_database.main.id
  name        = "pg_trgm"
}

# --- Branches: opt-in, they cost extra --------------------------------------

resource "appwrite_postgresql_branch" "preview" {
  count = var.create_branches ? 1 : 0

  database_id = appwrite_postgresql_database.main.id
  branch_id   = "preview"
  ttl         = 3600
}

resource "appwrite_mysql_branch" "preview" {
  count = var.create_branches ? 1 : 0

  database_id = appwrite_mysql_database.main.id
  branch_id   = "preview"
  ttl         = 3600
}

resource "appwrite_mongo_branch" "preview" {
  count = var.create_branches ? 1 : 0

  database_id = appwrite_mongo_database.main.id
  branch_id   = "preview"
  ttl         = 3600
}

# --- Custom backup destination: opt-in, needs real bucket credentials -------

resource "appwrite_postgresql_backup_storage" "offsite" {
  count = var.backup_storage == null ? 0 : 1

  database_id      = appwrite_postgresql_database.main.id
  storage_provider = var.backup_storage.storage_provider
  bucket           = var.backup_storage.bucket
  access_key       = var.backup_storage.access_key
  secret_key       = var.backup_storage.secret_key
  region           = var.backup_storage.region
  prefix           = var.backup_storage.prefix
  endpoint         = var.backup_storage.endpoint
}
