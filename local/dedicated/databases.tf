# One dedicated database per engine.
#
# Each of these provisions real, billable infrastructure and takes several
# minutes; the provider waits for the database to leave `provisioning` before
# the apply returns, so expect a slow first apply.

resource "appwrite_postgresql_database" "main" {
  id                   = "${var.prefix}-pg"
  name                 = "Sandbox PostgreSQL"
  specification        = local.postgresql_specification
  idle_timeout_minutes = var.idle_timeout_minutes

  maintenance_window_day      = "sun"
  maintenance_window_hour_utc = 3
}

resource "appwrite_mysql_database" "main" {
  id                   = "${var.prefix}-mysql"
  name                 = "Sandbox MySQL"
  specification        = local.mysql_specification
  idle_timeout_minutes = var.idle_timeout_minutes

  maintenance_window_day      = "sun"
  maintenance_window_hour_utc = 4
}

resource "appwrite_mongo_database" "main" {
  id                   = "${var.prefix}-mongo"
  name                 = "Sandbox MongoDB"
  specification        = local.mongo_specification
  idle_timeout_minutes = var.idle_timeout_minutes

  maintenance_window_day      = "sun"
  maintenance_window_hour_utc = 5
}
