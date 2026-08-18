# A dedicated PostgreSQL database runs on infrastructure reserved for one
# project. Creating one takes several minutes; Terraform waits for it to become
# ready so that resources depending on it get a database they can connect to.
resource "appwrite_postgresql_database" "main" {
  name          = "main"
  version       = "17"
  specification = "s-1vcpu-1gb"
}

# Pick the specification from the API instead of hardcoding a slug that may not
# be available on your billing plan.
data "appwrite_postgresql_specifications" "available" {}

resource "appwrite_postgresql_database" "production" {
  name          = "production"
  version       = "17"
  specification = one([for s in data.appwrite_postgresql_specifications.available.specifications : s.slug if s.enabled])

  # High availability: a warm replica with synchronous replication.
  replicas  = 1
  sync_mode = "sync"

  # Point-in-time recovery with a two week window.
  pitr                = true
  pitr_retention_days = 14

  # Grow storage automatically, up to 100 GB.
  storage_autoscaling                   = true
  storage_autoscaling_threshold_percent = 80
  storage_autoscaling_max_gb            = 100

  # Only reachable from the office range and the VPC.
  network_ip_allowlist = ["203.0.113.0/24", "10.0.0.0/16"]

  # Patch on Sunday mornings rather than mid-week.
  maintenance_window_day      = "sun"
  maintenance_window_hour_utc = 3
}

# A development database that scales to zero after 15 minutes of inactivity, and
# that can be paused entirely without losing its data.
resource "appwrite_postgresql_database" "development" {
  name                 = "development"
  specification        = "s-1vcpu-1gb"
  idle_timeout_minutes = 15
  status               = "ready"
}

# Run read-only SQL through the Appwrite API rather than opening a direct
# connection. Statement types beyond SELECT have to be opted into explicitly.
resource "appwrite_postgresql_database" "analytics" {
  name          = "analytics"
  specification = "s-1vcpu-1gb"

  sql_api_enabled            = true
  sql_api_allowed_statements = ["SELECT"]
  sql_api_max_rows           = 1000
  sql_api_timeout_seconds    = 30
}
