# This deployment has no shared pool for either product, so the API rejects a
# database created without a specification (`dedicated_database_required`).
# Reading the slugs here keeps the sandbox from hardcoding one that the billing
# plan may not allow, and the read costs nothing.
data "appwrite_documentsdb_specifications" "available" {}
data "appwrite_vectorsdb_specifications" "available" {}

locals {
  documentsdb_specification = coalesce(
    var.documentsdb_specification,
    try(split("|", sort([
      for s in data.appwrite_documentsdb_specifications.available.specifications :
      format("%012.2f|%s", s.price, s.slug) if s.enabled
    ])[0])[1], null),
  )
  vectorsdb_specification = coalesce(
    var.vectorsdb_specification,
    try(split("|", sort([
      for s in data.appwrite_vectorsdb_specifications.available.specifications :
      format("%012.2f|%s", s.price, s.slug) if s.enabled
    ])[0])[1], null),
  )
}

variable "documentsdb_specification" {
  description = "Compute slug for the DocumentsDB database. Null picks the cheapest enabled one."
  type        = string
  default     = null
}

variable "vectorsdb_specification" {
  description = "Compute slug for the VectorsDB database. Null picks the cheapest enabled one."
  type        = string
  default     = null
}
