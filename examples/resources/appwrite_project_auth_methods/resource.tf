# Email and password sign-in, team invitations and JWTs only.
resource "appwrite_project_auth_methods" "example" {
  email_password = true
  invites        = true
  jwt            = true

  magic_url = false
  email_otp = false
  anonymous = false
  phone     = false
}
