resource "appwrite_storage_bucket" "uploads" {
  name = "uploads"
}

resource "appwrite_storage_bucket" "images" {
  name                   = "images"
  maximum_file_size       = 10485760
  allowed_file_extensions = ["jpg", "png", "webp", "gif"]
  compression            = "gzip"
  transformations        = true
}

resource "appwrite_storage_bucket" "documents" {
  name         = "documents"
  file_security = true
  encryption   = true
  antivirus    = true
}

resource "appwrite_storage_file" "readme" {
  bucket_id = appwrite_storage_bucket.documents.id
  name      = "readme.txt"
  file_path = "files/readme.txt"
}
