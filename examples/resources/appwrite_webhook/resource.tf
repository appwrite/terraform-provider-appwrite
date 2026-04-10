resource "appwrite_webhook" "order_created" {
  name   = "Order Created"
  url    = "https://api.example.com/webhooks/orders"
  events = ["databases.*.collections.*.documents.*.create"]
}

resource "appwrite_webhook" "user_events" {
  name     = "User Events"
  url      = "https://api.example.com/webhooks/users"
  events   = ["users.*.create", "users.*.update", "users.*.delete"]
  security = true
}

resource "appwrite_webhook" "authenticated" {
  name      = "Authenticated Webhook"
  url       = "https://api.example.com/webhooks/secure"
  events    = ["databases.*.collections.*.documents.*.create"]
  http_user = "webhook"
  http_pass = var.webhook_password
  security  = true
}
