resource "appwrite_auth_team" "engineering" {
  name = "Engineering"
}

resource "appwrite_messaging_topic" "announcements" {
  name = "announcements"
}

# Topic restricted to a specific team
resource "appwrite_messaging_topic" "engineering_alerts" {
  name      = "engineering-alerts"
  subscribe = ["team:${appwrite_auth_team.engineering.id}"]
}
