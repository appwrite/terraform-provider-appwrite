# Anything not set here is kept at, or returned to, Appwrite's default.
resource "appwrite_project_auth_policies" "example" {
  password_dictionary    = true
  password_history       = 5
  password_personal_data = true

  password_strength = {
    min       = 12
    uppercase = true
    number    = true
  }

  session_duration = 86400 # one day
  session_limit    = 5

  mfa_factors = {
    phone = false
  }

  deny_disposable_email = true
}
