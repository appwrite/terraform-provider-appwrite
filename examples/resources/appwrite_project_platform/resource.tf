resource "appwrite_project_platform" "web" {
  type     = "web"
  name     = "Web app"
  hostname = "app.example.com"
}

# A leading wildcard allows every subdomain, such as preview deployments.
resource "appwrite_project_platform" "previews" {
  type     = "web"
  name     = "Preview deployments"
  hostname = "*.preview.example.com"
}

resource "appwrite_project_platform" "android" {
  type           = "android"
  name           = "Android app"
  application_id = "com.example.app"
}

resource "appwrite_project_platform" "ios" {
  type              = "apple"
  name              = "iOS app"
  bundle_identifier = "com.example.app"
}

resource "appwrite_project_platform" "linux" {
  type         = "linux"
  name         = "Linux app"
  package_name = "example-app"
}

resource "appwrite_project_platform" "windows" {
  type                    = "windows"
  name                    = "Windows app"
  package_identifier_name = "Example.App"
}
