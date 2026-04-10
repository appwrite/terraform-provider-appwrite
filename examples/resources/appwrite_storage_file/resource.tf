resource "appwrite_storage_bucket" "assets" {
  name = "assets"
}

resource "appwrite_storage_file" "logo" {
  bucket_id = appwrite_storage_bucket.assets.id
  name      = "logo.png"
  file_path = "assets/logo.png"
}

resource "appwrite_storage_file" "config" {
  bucket_id   = appwrite_storage_bucket.assets.id
  name        = "config.json"
  file_path   = "assets/config.json"
  permissions = ["read(\"any\")"]
}
