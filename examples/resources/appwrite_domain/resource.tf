resource "appwrite_domain" "example" {
  domain = "example.com"
}

# Delegate the domain to Appwrite at its registrar.
output "nameservers" {
  value = appwrite_domain.example.appwrite_nameservers
}
