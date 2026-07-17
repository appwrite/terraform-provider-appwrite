resource "appwrite_dedicated_database" "postgres" {
  engine        = "postgresql"
  name          = "analytics"
  version       = "16"
  specification = "s-1vcpu-2gb"
}

resource "appwrite_dedicated_database" "highly_available" {
  engine               = "mysql"
  name                 = "orders"
  specification        = "s-2vcpu-4gb"
  replicas             = 2
  sync_mode            = "sync"
  pitr                 = true
  pitr_retention_days  = 7
  storage_autoscaling  = true
  network_ip_allowlist = ["10.0.0.0/8"]
}

# Connection details are computed once the database finishes provisioning.
output "orders_connection_string" {
  value     = appwrite_dedicated_database.highly_available.connection_string
  sensitive = true
}
