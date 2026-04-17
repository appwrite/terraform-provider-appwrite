resource "appwrite_webhook" "order_events" {
  name   = "Order Events"
  url    = "https://api.example.com/webhooks/orders"
  events = ["databases.*.collections.*.documents.*.create"]
}

resource "appwrite_webhook" "user_events" {
  name     = "User Events"
  url      = "https://api.example.com/webhooks/users"
  events   = ["users.*.create", "users.*.update", "users.*.delete"]
  security = true
}
