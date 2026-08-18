resource "appwrite_postgresql_database" "main" {
  name          = "main"
  specification = "s-1vcpu-1gb"
}

# Send backups to a bucket you control, so they survive the Appwrite project and
# can be held under your own retention and compliance rules.
#
# Appwrite exposes no route to read this configuration back, so Terraform cannot
# detect drift here and cannot import an existing configuration. Destroying this
# resource only forgets it: backups keep going to whatever was last applied.
resource "appwrite_postgresql_backup_storage" "offsite" {
  database_id      = appwrite_postgresql_database.main.id
  storage_provider = "s3"
  bucket           = "acme-database-backups"
  region           = "eu-west-1"
  prefix           = "postgresql/main"

  access_key = var.backup_access_key
  secret_key = var.backup_secret_key
}

# An S3-compatible provider that is not Amazon needs an explicit endpoint.
resource "appwrite_postgresql_backup_storage" "r2" {
  database_id      = appwrite_postgresql_database.main.id
  storage_provider = "s3"
  bucket           = "acme-backups"
  endpoint         = "https://<account-id>.r2.cloudflarestorage.com"

  access_key = var.r2_access_key
  secret_key = var.r2_secret_key
}

variable "backup_access_key" {
  type      = string
  sensitive = true
}

variable "backup_secret_key" {
  type      = string
  sensitive = true
}

variable "r2_access_key" {
  type      = string
  sensitive = true
}

variable "r2_secret_key" {
  type      = string
  sensitive = true
}
