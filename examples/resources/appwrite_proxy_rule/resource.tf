resource "appwrite_site" "example" {
  name          = "example-site"
  framework     = "other"
  build_runtime = "node-22"
}

resource "appwrite_proxy_rule" "example" {
  domain      = "www.example.com"
  type        = "site"
  resource_id = appwrite_site.example.id
}
