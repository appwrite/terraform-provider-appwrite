resource "appwrite_site_variable" "api_url" {
  site_id = appwrite_site.dashboard.id
  key     = "NEXT_PUBLIC_API_URL"
  value   = "https://api.example.com"
}

resource "appwrite_site_variable" "secret_key" {
  site_id = appwrite_site.dashboard.id
  key     = "SECRET_KEY"
  value   = var.secret_key
  secret  = true
}
