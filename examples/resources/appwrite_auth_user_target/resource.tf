resource "appwrite_auth_user" "jane" {
  name  = "Jane Doe"
  email = "jane@example.com"
}

# A second address for Jane, delivered through a specific provider.
resource "appwrite_auth_user_target" "jane_alerts" {
  user_id       = appwrite_auth_user.jane.id
  provider_type = "email"
  identifier    = "jane.alerts@example.com"
  name          = "Alerts inbox"
  provider_id   = appwrite_messaging_provider.smtp.id
}

resource "appwrite_auth_user_target" "jane_phone" {
  user_id       = appwrite_auth_user.jane.id
  provider_type = "sms"
  identifier    = "+14155550100"
}

resource "appwrite_auth_user_target" "jane_device" {
  user_id       = appwrite_auth_user.jane.id
  provider_type = "push"
  identifier    = var.jane_device_token
  name          = "Jane's phone"
}

resource "appwrite_messaging_topic" "incidents" {
  name = "incidents"
}

resource "appwrite_messaging_subscriber" "jane_alerts" {
  topic_id  = appwrite_messaging_topic.incidents.id
  target_id = appwrite_auth_user_target.jane_alerts.id
}
