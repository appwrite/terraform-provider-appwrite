provider "appwrite" {
  endpoint   = "https://cloud.appwrite.io/v1"
  project_id = "project-id"
  api_key    = "api-key"
}

# For Appwrite Community Edition with self-signed certificates:
#
# provider "appwrite" {
#   endpoint    = "https://appwrite-instance.com/v1"
#   project_id  = "project-id"
#   api_key     = "api-key"
#   self_signed = true
# }
