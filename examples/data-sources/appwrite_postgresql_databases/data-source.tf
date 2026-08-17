data "appwrite_postgresql_databases" "all" {}

# Filter server-side with Appwrite query strings rather than pulling everything
# back and filtering in HCL.
data "appwrite_postgresql_databases" "ready" {
  queries = [
    "equal(\"status\", \"ready\")",
    "orderDesc(\"$createdAt\")",
  ]
}

output "database_names" {
  value = [for d in data.appwrite_postgresql_databases.all.databases : d.name]
}

# Connection credentials are deliberately absent here; read them from the
# singular data source for the one database that needs them, so a listing does
# not put every password into state.
output "undersized" {
  value = [
    for d in data.appwrite_postgresql_databases.all.databases :
    d.name if d.memory < 2048
  ]
}
