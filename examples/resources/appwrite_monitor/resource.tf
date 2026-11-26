resource "appwrite_monitor" "api" {
  type = "http"
  name = "API health"
  url  = "https://api.example.com/health"

  headers = {
    Authorization = "Bearer ${var.health_token}"
  }

  rules = [
    jsonencode({ method = "notBetween", attribute = "statusCode", values = [200, 299] }),
    jsonencode({ method = "greaterThan", attribute = "latency", values = [2000] }),
  ]

  interval      = 60
  severity      = "critical"
  escalation_id = appwrite_escalation.on_call.id
}

resource "appwrite_monitor" "database" {
  type = "tcp"
  name = "Database port"
  host = "db.example.com"
  port = 5432
}

resource "appwrite_monitor" "bandwidth" {
  type = "usage"
  name = "Bandwidth over 5 GB a day"

  rules = [
    jsonencode({ method = "greaterThan", attribute = "network.outbound", values = [5000000000] }),
  ]

  window = 86400
  notify = ["owner", "developer"]
}

variable "health_token" {
  type      = string
  sensitive = true
}
