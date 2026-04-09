resource "appwrite_bucket" "uploads" {
  id   = "uploads"
  name = "uploads"
}

resource "appwrite_bucket" "images" {
  id                     = "images"
  name                   = "images"
  maximum_file_size       = 10485760
  allowed_file_extensions = ["jpg", "png", "webp", "gif"]
  compression            = "gzip"
  transformations        = true
}

resource "appwrite_bucket" "documents" {
  id           = "documents"
  name         = "documents"
  file_security = true
  encryption   = true
  antivirus    = true
}
