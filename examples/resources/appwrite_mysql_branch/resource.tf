resource "appwrite_mysql_database" "main" {
  name          = "main"
  specification = "db-s-1vcpu-1gb"
}

# A branch is a copy of the parent that shares its credentials but has its own
# host and database name.
resource "appwrite_mysql_branch" "preview" {
  database_id = appwrite_mysql_database.main.id
  branch_id   = "preview"
}

# A branch with a TTL is deleted by the server when it expires. Terraform drops
# it from state on the next refresh and recreates it on the following apply.
resource "appwrite_mysql_branch" "ephemeral" {
  database_id = appwrite_mysql_database.main.id
  branch_id   = "ci-run"
  ttl         = 3600
}

output "preview_host" {
  value = appwrite_mysql_branch.preview.host
}
