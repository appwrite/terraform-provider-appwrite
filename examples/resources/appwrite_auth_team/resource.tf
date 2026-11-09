resource "appwrite_auth_team" "engineering" {
  name = "engineering"
}

resource "appwrite_auth_team" "marketing" {
  name  = "marketing"
  roles = ["owner", "editor"]
}

# Preferences shared by every member. Leave prefs out to keep whatever the
# team already has.
resource "appwrite_auth_team" "support" {
  name = "support"
  prefs = jsonencode({
    timezone = "Europe/Amsterdam"
    sla      = { response_hours = 4 }
  })
}
