# Sendgrid email provider
resource "appwrite_messaging_provider" "sendgrid" {
  name       = "sendgrid"
  type       = "sendgrid"
  api_key    = var.sendgrid_api_key
  from_email = "noreply@example.com"
  from_name  = "application"
}

# SMTP email provider
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

# Twilio SMS provider
resource "appwrite_messaging_provider" "twilio" {
  name        = "twilio"
  type        = "twilio"
  account_sid = var.twilio_account_sid
  auth_token  = var.twilio_auth_token
  from        = "+1234567890"
}

# FCM push notification provider
resource "appwrite_messaging_provider" "fcm" {
  name                 = "fcm"
  type                 = "fcm"
  service_account_json = file("firebase-service-account.json")
}
