data "appwrite_storage_bucket" "uploads" {
  id = "uploads"
}

output "bucket_name" {
  value = data.appwrite_storage_bucket.uploads.name
}
