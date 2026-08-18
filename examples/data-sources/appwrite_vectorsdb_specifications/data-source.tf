data "appwrite_vectorsdb_specifications" "available" {}

# A deployment without a shared pool rejects a database created with no
# specification, so reading the slugs here avoids hardcoding one the billing
# plan may not allow.
output "available_specifications" {
  value = [for s in data.appwrite_vectorsdb_specifications.available.specifications : s.slug if s.enabled]
}
