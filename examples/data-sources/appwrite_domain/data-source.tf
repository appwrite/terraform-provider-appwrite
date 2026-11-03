data "appwrite_domain" "example" {
  domain = "example.com"
}

resource "appwrite_domain_record" "www" {
  domain_id = data.appwrite_domain.example.id
  type      = "CNAME"
  name      = "www"
  value     = "example.appwrite.network"
}
