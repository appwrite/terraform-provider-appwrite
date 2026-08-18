data "appwrite_mongo_specifications" "available" {}

output "available_specifications" {
  value = [for s in data.appwrite_mongo_specifications.available.specifications : s.slug if s.enabled]
}
