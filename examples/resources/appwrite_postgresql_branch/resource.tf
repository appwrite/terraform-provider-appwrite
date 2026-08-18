resource "appwrite_postgresql_database" "main" {
  name          = "main"
  specification = "s-1vcpu-1gb"
}

# A branch is a copy of the parent that shares its credentials but has its own
# host and database name. Useful for giving a preview environment real data
# without pointing it at production.
resource "appwrite_postgresql_branch" "preview" {
  database_id = appwrite_postgresql_database.main.id
  branch_id   = "preview"
}

# A branch with a TTL is deleted by the server when it expires. Terraform drops
# it from state on the next refresh and recreates it on the following apply, so
# expect a short-lived branch to come back rather than stay gone.
resource "appwrite_postgresql_branch" "ephemeral" {
  database_id = appwrite_postgresql_database.main.id
  branch_id   = "ci-run"
  ttl         = 3600
}

# Branches have no update route, so changing any argument replaces the branch
# and discards whatever was written to it.
output "preview_host" {
  value = appwrite_postgresql_branch.preview.host
}

output "preview_connection_string" {
  value     = appwrite_postgresql_branch.preview.connection_string
  sensitive = true
}
