data "appwrite_domain" "registered" {
  domain = "example.com"
}

# Delegate a domain registered through Appwrite to another DNS provider.
resource "appwrite_domain_nameservers" "example" {
  domain_id   = data.appwrite_domain.registered.id
  nameservers = ["ns1.dns.example.net", "ns2.dns.example.net"]
}
