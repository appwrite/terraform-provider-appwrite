data "appwrite_mongo_backups" "main" {
  database_id = "main"

  queries = [
    "equal(\"status\", \"completed\")",
    "orderDesc(\"$createdAt\")",
  ]
}

# Restoring is not a Terraform operation, so this is how a backup ID is found
# for a restore run through the Console or API.
output "latest_backup_id" {
  value = try(data.appwrite_mongo_backups.main.backups[0].id, null)
}
