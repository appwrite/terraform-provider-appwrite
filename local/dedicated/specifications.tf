# Read the available compute specifications before creating anything. These are
# plain reads, so `terraform plan` resolves them without provisioning, and the
# plan output shows which slug each database would actually use.
#
# Which slugs exist depends on the organization's billing plan, so hardcoding
# one risks an apply that fails minutes in. Filtering on `enabled` avoids that.
data "appwrite_postgresql_specifications" "available" {}
data "appwrite_mysql_specifications" "available" {}
data "appwrite_mongo_specifications" "available" {}

locals {
  # Sort by price, zero-padded so string ordering matches numeric ordering, and
  # take the cheapest slug the plan allows.
  postgresql_candidates = sort([
    for s in data.appwrite_postgresql_specifications.available.specifications :
    format("%012.2f|%s", s.price, s.slug) if s.enabled
  ])
  mysql_candidates = sort([
    for s in data.appwrite_mysql_specifications.available.specifications :
    format("%012.2f|%s", s.price, s.slug) if s.enabled
  ])
  mongo_candidates = sort([
    for s in data.appwrite_mongo_specifications.available.specifications :
    format("%012.2f|%s", s.price, s.slug) if s.enabled
  ])

  postgresql_specification = coalesce(
    var.postgresql_specification,
    try(split("|", local.postgresql_candidates[0])[1], null),
  )
  mysql_specification = coalesce(
    var.mysql_specification,
    try(split("|", local.mysql_candidates[0])[1], null),
  )
  mongo_specification = coalesce(
    var.mongo_specification,
    try(split("|", local.mongo_candidates[0])[1], null),
  )
}
