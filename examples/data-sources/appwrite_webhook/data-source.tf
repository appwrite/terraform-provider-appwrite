data "appwrite_webhook" "slack" {
  id = "64f2cd7e27bda9f23ab6"
}

output "webhook_url" {
  value = data.appwrite_webhook.slack.url
}
