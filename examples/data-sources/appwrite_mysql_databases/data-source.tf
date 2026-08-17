data "appwrite_mysql_databases" "all" {}

data "appwrite_mysql_databases" "ready" {
  queries = ["equal(\"status\", \"ready\")"]
}

output "database_names" {
  value = [for d in data.appwrite_mysql_databases.all.databases : d.name]
}
