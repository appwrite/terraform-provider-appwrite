resource "appwrite_mongo_database" "main" {
  name          = "main"
  specification = "s-1vcpu-1gb"
}

resource "appwrite_mongo_database" "production" {
  name          = "production"
  specification = "s-2vcpu-4gb"

  replicas  = 2
  sync_mode = "quorum"

  pitr                = true
  pitr_retention_days = 30

  storage_autoscaling                   = true
  storage_autoscaling_threshold_percent = 75
  storage_autoscaling_max_gb            = 500

  maintenance_window_day      = "sat"
  maintenance_window_hour_utc = 2
}
