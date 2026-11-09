resource "appwrite_site" "example" {
  name          = "example-site"
  framework     = "other"
  build_runtime = "node-22"
}

# Serve a site on a custom domain.
resource "appwrite_proxy_rule" "example" {
  domain      = "www.example.com"
  type        = "site"
  resource_id = appwrite_site.example.id
}

# Serve the Appwrite API on a custom domain.
resource "appwrite_proxy_rule" "api" {
  domain = "api.example.com"
  type   = "api"
}

# Redirect the apex domain to the site, listed under the site's domains.
resource "appwrite_proxy_rule" "apex" {
  domain                 = "example.com"
  type                   = "redirect"
  redirect_url           = "https://www.example.com"
  redirect_status_code   = 301
  resource_id            = appwrite_site.example.id
  redirect_resource_type = "site"
}
