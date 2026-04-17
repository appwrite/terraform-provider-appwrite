terraform {
  required_providers {
    appwrite = {
      source = "appwrite/appwrite"
    }
  }
}

# Configure the provider using environment variables:
#
#   export APPWRITE_ENDPOINT="https://cloud.appwrite.io/v1"
#   export APPWRITE_PROJECT_ID="project-id"
#   export APPWRITE_API_KEY="api-key"
#
#   or:
#
#   provider "appwrite" {
#     endpoint   = "https://cloud.appwrite.io/v1"
#     project_id = "project-id"
#     api_key    = "api-key"
#     self_signed = true
#   }

provider "appwrite" {}
