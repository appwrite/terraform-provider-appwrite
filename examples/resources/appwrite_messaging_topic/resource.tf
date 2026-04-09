resource "appwrite_messaging_topic" "announcements" {
  name = "announcements"
}

resource "appwrite_messaging_topic" "alerts" {
  name      = "alerts"
  subscribe = ["users"]
}
