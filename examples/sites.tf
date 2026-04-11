resource "appwrite_site" "landing_page" {
  name          = "landing-page"
  framework     = "other"
  build_runtime = "node-22"
}

resource "appwrite_site" "dashboard" {
  name            = "dashboard"
  framework       = "nextjs"
  build_runtime   = "node-22"
  install_command = "npm install"
  build_command   = "npm run build"
}

resource "appwrite_site_variable" "dashboard_api_url" {
  site_id = appwrite_site.dashboard.id
  key     = "NEXT_PUBLIC_API_URL"
  value   = "https://api.example.com"
}

resource "appwrite_site_variable" "landing_ga_id" {
  site_id = appwrite_site.landing_page.id
  key     = "GA_ID"
  value   = "G-EXAMPLE123"
}
