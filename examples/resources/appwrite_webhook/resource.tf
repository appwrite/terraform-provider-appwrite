resource "appwrite_webhook" "order_created" {
  name   = "order created"
  url    = "https://api.example.com/webhooks/orders"
  events = ["databases.*.collections.*.documents.*.create"]
}

resource "appwrite_webhook" "user_events" {
  name   = "user events"
  url    = "https://api.example.com/webhooks/users"
  events = ["users.*.create", "users.*.update", "users.*.delete"]
  tls    = true
}

resource "appwrite_webhook" "authenticated" {
  name          = "authenticated webhook"
  url           = "https://api.example.com/webhooks/secure"
  events        = ["databases.*.collections.*.documents.*.create"]
  auth_username = "webhook"
  auth_password = var.webhook_password
  tls           = true
}
