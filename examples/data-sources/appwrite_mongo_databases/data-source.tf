data "appwrite_mongo_databases" "all" {}

data "appwrite_mongo_databases" "ready" {
  queries = ["equal(\"status\", \"ready\")"]
}

output "database_names" {
  value = [for d in data.appwrite_mongo_databases.all.databases : d.name]
}
