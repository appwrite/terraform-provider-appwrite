resource "appwrite_auth_team" "engineering" {
  name = "Engineering"
}

resource "appwrite_auth_user" "jane" {
  name  = "Jane Doe"
  email = "jane@example.com"
}

# An existing user, by ID.
resource "appwrite_auth_team_membership" "jane" {
  team_id = appwrite_auth_team.engineering.id
  user_id = appwrite_auth_user.jane.id
  roles   = ["owner", "developer"]
}

# By email address. Appwrite adds the user with this address, or creates one
# named `name` if there is none.
resource "appwrite_auth_team_membership" "contractor" {
  team_id = appwrite_auth_team.engineering.id
  email   = "contractor@example.com"
  name    = "Contractor"
  roles   = ["developer"]
}
