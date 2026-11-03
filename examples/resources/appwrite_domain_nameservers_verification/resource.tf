resource "appwrite_domain" "example" {
  domain = "example.com"
}

# Set appwrite_domain.example.appwrite_nameservers at the domain's registrar,
# for example with that registrar's own Terraform provider, then wait for
# Appwrite to see the delegation.
resource "appwrite_domain_nameservers_verification" "example" {
  domain_id = appwrite_domain.example.id

  timeouts {
    create = "2h"
  }
}

# Records that only work once Appwrite serves the zone can wait for it.
resource "appwrite_domain_record" "www" {
  domain_id = appwrite_domain_nameservers_verification.example.id
  type      = "CNAME"
  name      = "www"
  value     = "example.appwrite.network"
}
