resource "appwrite_messaging_topic" "announcements" {
  name = "announcements"
}

resource "appwrite_messaging_provider" "sendgrid" {
  name       = "sendgrid"
  type       = "sendgrid"
  api_key    = "SG.test"
  from_email = "noreply@example.com"
  from_name  = "application"
}
