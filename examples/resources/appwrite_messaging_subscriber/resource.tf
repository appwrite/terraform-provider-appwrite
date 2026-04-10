resource "appwrite_messaging_topic" "announcements" {
  name = "announcements"
}

resource "appwrite_messaging_subscriber" "user_email" {
  topic_id  = appwrite_messaging_topic.announcements.id
  target_id = "user-email-target-id"
}
