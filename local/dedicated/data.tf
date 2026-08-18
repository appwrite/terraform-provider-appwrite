# Read back what was created. These depend on the resources, so they refresh
# after the databases exist rather than failing on the first plan.

data "appwrite_postgresql_extensions" "main" {
  database_id = appwrite_postgresql_database.main.id
}

data "appwrite_postgresql_database_status" "main" {
  database_id = appwrite_postgresql_database.main.id
}

data "appwrite_mysql_database_status" "main" {
  database_id = appwrite_mysql_database.main.id
}

data "appwrite_mongo_database_status" "main" {
  database_id = appwrite_mongo_database.main.id
}

# Everything the project has, so a leftover sandbox database from an earlier run
# is easy to spot.
data "appwrite_postgresql_databases" "all" {}
data "appwrite_mysql_databases" "all" {}
data "appwrite_mongo_databases" "all" {}
