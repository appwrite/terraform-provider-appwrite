data "appwrite_health" "status" {}

output "server_status" {
  value = data.appwrite_health.status.status
}

output "server_ping" {
  value = data.appwrite_health.status.ping
}
