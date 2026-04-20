data "appwrite_site" "landing" {
  id = "64f2cd7e27bda9f23ab6"
}

output "site_framework" {
  value = data.appwrite_site.landing.framework
}
