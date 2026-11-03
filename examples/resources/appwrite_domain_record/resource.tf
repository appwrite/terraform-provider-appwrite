resource "appwrite_domain_record" "www" {
  domain_id = appwrite_domain.example.id
  type      = "A"
  name      = "www"
  value     = "203.0.113.10"
  ttl       = 300
}

resource "appwrite_domain_record" "mail" {
  domain_id = appwrite_domain.example.id
  type      = "MX"
  name      = "@"
  value     = "mx1.mail.example.net"
  priority  = 10
}

resource "appwrite_domain_record" "sip" {
  domain_id = appwrite_domain.example.id
  type      = "SRV"
  name      = "_sip._tcp"
  value     = "sip.example.net"
  priority  = 10
  weight    = 5
  port      = 5060
}

resource "appwrite_domain_record" "spf" {
  domain_id = appwrite_domain.example.id
  type      = "TXT"
  name      = "@"
  value     = "v=spf1 include:mail.example.net -all"
  comment   = "Mail provider SPF"
}
