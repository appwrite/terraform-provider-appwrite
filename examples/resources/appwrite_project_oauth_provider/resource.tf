resource "appwrite_project_oauth_provider" "github" {
  provider_id              = "github"
  client_id                = "Iv1.0123456789abcdef"
  client_secret_wo         = var.github_client_secret
  client_secret_wo_version = 1
}

resource "appwrite_project_oauth_provider" "google" {
  provider_id      = "google"
  client_id        = "1234567890-abc.apps.googleusercontent.com"
  client_secret_wo = var.google_client_secret
  # Increment after rotating the secret in Google Cloud.
  client_secret_wo_version = 1
  prompt                   = ["select_account"]
}

# A generic OpenID Connect identity provider, discovered from its
# well-known configuration.
resource "appwrite_project_oauth_provider" "oidc" {
  provider_id              = "oidc"
  client_id                = "appwrite"
  client_secret_wo         = var.oidc_client_secret
  client_secret_wo_version = 1
  well_known_url           = "https://login.example.com/.well-known/openid-configuration"
}

# Sign in with Apple takes the Services ID as client_id and the contents of
# the .p8 key file as the secret.
resource "appwrite_project_oauth_provider" "apple" {
  provider_id              = "apple"
  client_id                = "com.example.web"
  client_secret_wo         = file("${path.module}/AuthKey_P4000000N8.p8")
  client_secret_wo_version = 1
  key_id                   = "P4000000N8"
  team_id                  = "D4000000R6"
}
