data "appwrite_mysql_specifications" "available" {}

output "available_specifications" {
  value = [for s in data.appwrite_mysql_specifications.available.specifications : s.slug if s.enabled]
}
