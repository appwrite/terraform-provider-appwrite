resource "appwrite_webhook" "order_created" {
  name   = "order created"
  url    = "https://api.example.com/webhooks/orders"
  events = ["databases.*.collections.*.documents.*.create"]
}

resource "appwrite_webhook" "user_events" {
  name     = "user events"
  url      = "https://api.example.com/webhooks/users"
  events   = ["users.*.create", "users.*.update", "users.*.delete"]
  security = true
}

resource "appwrite_webhook" "authenticated" {
  name      = "authenticated webhook"
  url       = "https://api.example.com/webhooks/secure"
  events    = ["databases.*.collections.*.documents.*.create"]
  http_user = "webhook"
  http_pass = var.webhook_password
  security  = true
}
