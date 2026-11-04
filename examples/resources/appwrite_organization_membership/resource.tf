# Add a member by email address. If no Appwrite account uses it, Appwrite
# creates one.
resource "appwrite_organization_membership" "jane" {
  email = "jane@example.com"
  roles = ["developer"]
}

# Add an existing account by its user ID.
resource "appwrite_organization_membership" "ops" {
  user_id = "6a3f1c2b0011aabbccdd"
  roles   = ["owner"]
}
