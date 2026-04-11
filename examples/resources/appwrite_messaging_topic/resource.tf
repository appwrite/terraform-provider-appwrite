resource "appwrite_messaging_topic" "announcements" {
  name = "announcements"
}

resource "appwrite_auth_team" "engineering" {
  name = "engineering"
}

resource "appwrite_messaging_topic" "engineering_alerts" {
  name      = "engineering-alerts"
  subscribe = ["team:${appwrite_auth_team.engineering.id}"]
}
