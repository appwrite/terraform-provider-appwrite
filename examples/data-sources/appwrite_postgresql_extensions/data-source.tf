data "appwrite_postgresql_extensions" "main" {
  database_id = "main"
}

output "installed_extensions" {
  value = data.appwrite_postgresql_extensions.main.installed
}

# The metadata list explains what each installable extension provides.
output "extension_catalog" {
  value = {
    for e in data.appwrite_postgresql_extensions.main.metadata :
    e.key => "${e.name} (${e.category}): ${e.description}"
  }
}
