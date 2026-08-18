data "appwrite_postgresql_backups" "main" {
  database_id = "main"

  queries = [
    "equal(\"status\", \"completed\")",
    "orderDesc(\"$createdAt\")",
    "limit(10)",
  ]
}

# Restoring is not a Terraform operation, so this is how a backup ID is found
# for a restore run through the Console or API.
output "latest_backup_id" {
  value = try(data.appwrite_postgresql_backups.main.backups[0].id, null)
}

output "recent_backups" {
  value = [
    for b in data.appwrite_postgresql_backups.main.backups : {
      id           = b.id
      type         = b.type
      size_bytes   = b.size_bytes
      completed_at = b.completed_at
    }
  ]
}

# A backup whose type differs from what was requested fell back; fallback_reason
# explains why, and is worth surfacing rather than silently accepting.
output "fell_back" {
  value = [
    for b in data.appwrite_postgresql_backups.main.backups :
    "${b.id}: wanted ${b.requested_type}, got ${b.type} (${b.fallback_reason})"
    if b.fallback_reason != ""
  ]
}
