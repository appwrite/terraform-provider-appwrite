resource "appwrite_analytics_property" "website" {
  name            = "Website"
  domain          = "example.com"
  allowed_origins = ["https://example.com"]
}

resource "appwrite_analytics_property" "app" {
  id   = "app"
  name = "Mobile app"
  type = "mobile"
}
