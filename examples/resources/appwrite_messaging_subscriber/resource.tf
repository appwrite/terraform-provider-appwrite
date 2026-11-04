resource "appwrite_messaging_topic" "announcements" {
  name = "announcements"
}

resource "appwrite_user_target" "jane_newsletter" {
  user_id       = appwrite_auth_user.jane.id
  provider_type = "email"
  identifier    = "jane.newsletter@example.com"
}

resource "appwrite_messaging_subscriber" "jane_newsletter" {
  topic_id  = appwrite_messaging_topic.announcements.id
  target_id = appwrite_user_target.jane_newsletter.id
}
