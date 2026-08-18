data "appwrite_postgresql_specifications" "available" {}

# Which slugs the plan actually allows is a server-side decision, so filter on
# `enabled` rather than assuming a slug exists.
output "available_specifications" {
  value = [
    for s in data.appwrite_postgresql_specifications.available.specifications : {
      slug   = s.slug
      cpu    = s.cpu
      memory = s.memory
      price  = s.price
    } if s.enabled
  ]
}

# Provision on the smallest specification the plan allows, so a slug that was
# withdrawn or is not included in the plan fails the plan instead of the apply.
locals {
  postgresql_specifications_by_memory = sort([
    for s in data.appwrite_postgresql_specifications.available.specifications :
    format("%08d|%s", s.memory, s.slug) if s.enabled
  ])
  smallest_postgresql_specification = split("|", local.postgresql_specifications_by_memory[0])[1]
}

resource "appwrite_postgresql_database" "main" {
  name          = "main"
  specification = local.smallest_postgresql_specification
}
