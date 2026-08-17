resource "appwrite_mysql_database" "main" {
  name          = "main"
  version       = "8"
  specification = "db-s-1vcpu-1gb"
}

resource "appwrite_mysql_database" "production" {
  name          = "production"
  specification = "db-s-2vcpu-4gb"

  replicas  = 1
  sync_mode = "sync"

  pitr                = true
  pitr_retention_days = 14

  storage_autoscaling                   = true
  storage_autoscaling_threshold_percent = 80
  storage_autoscaling_max_gb            = 200

  network_ip_allowlist = ["10.0.0.0/16"]

  maintenance_window_day      = "sun"
  maintenance_window_hour_utc = 4
}
