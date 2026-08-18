variable "self_signed" {
  description = "Accept self-signed certificates. Needed for some non-production endpoints."
  type        = bool
  default     = false
}

variable "prefix" {
  description = "Prefix for every database ID, so a sandbox run is easy to spot and clean up."
  type        = string
  default     = "tf-sandbox"
}

# Leave these null to let the config pick the cheapest specification the billing
# plan allows, read live from the API. Set one only to pin a specific slug.
variable "postgresql_specification" {
  description = "Compute slug for the PostgreSQL database. Null picks the cheapest enabled one."
  type        = string
  default     = null
}

variable "mysql_specification" {
  description = "Compute slug for the MySQL database. Null picks the cheapest enabled one."
  type        = string
  default     = null
}

variable "mongo_specification" {
  description = "Compute slug for the MongoDB database. Null picks the cheapest enabled one."
  type        = string
  default     = null
}

variable "idle_timeout_minutes" {
  description = "Minutes of inactivity before a database scales to zero. Keeps a sandbox cheap; 0 means always on."
  type        = number
  default     = 15
}

variable "create_branches" {
  description = "Also create a short-lived branch of each database. Branches cost extra."
  type        = bool
  default     = false
}

variable "backup_storage" {
  description = <<-EOT
    Optional custom backup destination, applied to all three databases. Leave null to use
    Appwrite's default storage. The API has no route to read this back, so Terraform cannot
    detect drift on it and destroying it only forgets it.
  EOT
  type = object({
    storage_provider = string
    bucket           = string
    access_key       = string
    secret_key       = string
    region           = optional(string)
    prefix           = optional(string)
    endpoint         = optional(string)
  })
  default   = null
  sensitive = true
}
