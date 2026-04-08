resource "appwrite_messaging_topic" "announcements" {
  id   = "announcements"
  name = "announcements"
}

resource "appwrite_messaging_topic" "alerts" {
  id        = "alerts"
  name      = "alerts"
  subscribe = ["users"]
}
