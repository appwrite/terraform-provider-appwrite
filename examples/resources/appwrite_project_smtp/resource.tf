variable "smtp_password" {
  type      = string
  sensitive = true
  ephemeral = true
}

resource "appwrite_project_smtp" "example" {
  host                = "smtp.example.com"
  port                = 587
  secure              = "tls"
  username            = "apikey"
  password_wo         = var.smtp_password
  password_wo_version = 1
  sender_email        = "noreply@example.com"
  sender_name         = "Example"
  reply_to_email      = "support@example.com"
}
