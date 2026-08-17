resource "appwrite_mongo_database" "main" {
  name          = "main"
  specification = "db-s-1vcpu-1gb"
}

# Appwrite exposes no route to read this configuration back, so Terraform cannot
# detect drift here and cannot import an existing configuration. Destroying this
# resource only forgets it: backups keep going to whatever was last applied.
resource "appwrite_mongo_backup_storage" "offsite" {
  database_id      = appwrite_mongo_database.main.id
  storage_provider = "s3"
  bucket           = "acme-database-backups"
  region           = "eu-west-1"
  prefix           = "mongo/main"

  access_key = var.backup_access_key
  secret_key = var.backup_secret_key
}

variable "backup_access_key" {
  type      = string
  sensitive = true
}

variable "backup_secret_key" {
  type      = string
  sensitive = true
}
