resource "appwrite_messaging_provider" "sendgrid" {
  name       = "sendgrid"
  type       = "sendgrid"
  api_key    = var.sendgrid_api_key
  from_email = "noreply@example.com"
  from_name  = "my app"
}

resource "appwrite_messaging_provider" "smtp" {
  name       = "smtp"
  type       = "smtp"
  host       = "smtp.example.com"
  port       = 587
  username   = "user@example.com"
  password   = var.smtp_password
  encryption = "tls"
  from_email = "noreply@example.com"
}

resource "appwrite_messaging_provider" "twilio" {
  name        = "twilio"
  type        = "twilio"
  account_sid = var.twilio_account_sid
  auth_token  = var.twilio_auth_token
  from        = "+1234567890"
}

resource "appwrite_messaging_provider" "fcm" {
  name                 = "fcm"
  type                 = "fcm"
  service_account_json = file("firebase-service-account.json")
}

resource "appwrite_messaging_provider" "ses" {
  name       = "ses"
  type       = "ses"
  access_key = var.aws_access_key_id
  secret_key = var.aws_secret_access_key
  region     = "us-east-1"
  from_email = "noreply@example.com"
  from_name  = "my app"
}

# Push messages delivered over Appwrite's own MQTT broker. Needs Appwrite 2.3.0
# or later.
resource "appwrite_messaging_provider" "appwrite" {
  name   = "appwrite push"
  type   = "appwrite"
  qos    = 1
  expiry = 86400
}
