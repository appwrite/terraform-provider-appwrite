# Let the office network through before any other rule is evaluated.
resource "appwrite_waf_rule" "office" {
  type     = "bypass"
  name     = "Office network"
  priority = -100
  conditions = [
    jsonencode({ method = "equal", attribute = "ip", values = ["198.51.100.0/24"] }),
  ]
}

resource "appwrite_waf_rule" "block_scrapers" {
  id          = "block-scrapers"
  type        = "deny"
  name        = "Block scrapers"
  description = "Known scraper user agents"
  conditions = [
    jsonencode({ method = "contains", attribute = "userAgent", values = ["python-requests", "scrapy"] }),
  ]
}

# Every condition must match. Nest conditions in "or" to match any of them.
resource "appwrite_waf_rule" "sign_in_limit" {
  type     = "rateLimit"
  name     = "Sign-in attempts"
  limit    = 10
  interval = 60
  strategy = "slidingWindow"
  conditions = [
    jsonencode({ method = "equal", attribute = "method", values = ["POST"] }),
    jsonencode({
      method = "or"
      values = [
        { method = "startsWith", attribute = "path", values = ["/v1/account/sessions"] },
        { method = "startsWith", attribute = "path", values = ["/v1/account/tokens"] },
      ]
    }),
  ]
}

resource "appwrite_waf_rule" "challenge_outside_eu" {
  type       = "challenge"
  name       = "Challenge traffic from outside Europe"
  difficulty = 2
  conditions = [
    jsonencode({ method = "notEqual", attribute = "continent", values = ["EU"] }),
  ]
}
