terraform {
  required_version = ">= 1.3"

  required_providers {
    appwrite = {
      source = "appwrite/appwrite"
    }
  }
}

# Configuration comes from the environment so no credentials land in a file:
#
#   export APPWRITE_ENDPOINT="https://<host>/v1"
#   export APPWRITE_PROJECT_ID="<project-id>"
#   export APPWRITE_API_KEY="<standard project key>"
#
provider "appwrite" {}

variable "prefix" {
  description = "Prefix for every ID, so a sandbox run is easy to spot and clean up."
  type        = string
  default     = "tf-sandbox"
}

variable "manage_collections" {
  description = <<-EOT
    Also manage collections, indexes and documents. Needs an API key with the
    collections.write and documents.write scopes; the database resources alone
    need only databases.write.
  EOT
  type        = bool
  default     = true
}

variable "dimension" {
  description = "Embedding dimension for the VectorsDB collection. Small keeps the example readable."
  type        = number
  default     = 4
}
