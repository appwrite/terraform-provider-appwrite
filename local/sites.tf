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

# Deploy a site from a code upload
resource "appwrite_site_deployment" "landing_page_code" {
  site_id     = appwrite_site.landing_page.id
  source_type = "code"
  code_path   = "./dist.tar.gz"
  code_hash   = filesha256("./files/dist.tar.gz")
  activate    = true
}

# Deploy a site from VCS (requires the site to have a VCS provider configured
# via installation_id, provider_repository_id, and provider_branch)
# resource "appwrite_site_deployment" "dashboard_vcs" {
#   site_id     = appwrite_site.dashboard.id
#   source_type = "vcs"
#   type        = "branch"
#   reference   = "main"
#   activate    = true
# }

# Deploy a site from a template
resource "appwrite_site_deployment" "dashboard_template" {
  site_id        = appwrite_site.dashboard.id
  source_type    = "template"
  repository     = "templates-for-sites"
  owner          = "appwrite"
  root_directory = "nextjs/starter"
  type           = "branch"
  reference      = "main"
  activate       = true
}
