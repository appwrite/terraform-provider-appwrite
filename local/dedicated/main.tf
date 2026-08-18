terraform {
  # optional() in an object type constraint needs 1.3+.
  required_version = ">= 1.3"

  required_providers {
    appwrite = {
      source = "appwrite/appwrite"
    }
  }
}

# Configuration comes from the environment so no credentials land in a file:
#
#   export APPWRITE_ENDPOINT="https://<staging-host>/v1"
#   export APPWRITE_PROJECT_ID="<project-id>"
#   export APPWRITE_API_KEY="<standard project key>"
#
provider "appwrite" {
  self_signed = var.self_signed
}
