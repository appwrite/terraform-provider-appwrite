resource "appwrite_topic" "announcements" {
  id   = "announcements"
  name = "announcements"
}

resource "appwrite_topic" "alerts" {
  id        = "alerts"
  name      = "alerts"
  subscribe = ["users"]
}
