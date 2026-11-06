# Shared by every function and site in the project.
resource "appwrite_project_variable" "api_url" {
  key    = "API_URL"
  value  = "https://api.example.com"
  secret = false
}

# Secret unless secret = false. With value_wo the value never reaches state;
# change value_wo_version to apply a new one. Requires Terraform 1.11 or later.
resource "appwrite_project_variable" "stripe_key" {
  key              = "STRIPE_SECRET_KEY"
  value_wo         = var.stripe_secret_key
  value_wo_version = 1
}
