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

# Subscriber requires a valid user target ID
# resource "appwrite_messaging_subscriber" "admin_announcements" {
#   topic_id  = appwrite_messaging_topic.announcements.id
#   target_id = appwrite_auth_user.john.id
# }
