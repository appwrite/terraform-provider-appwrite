# Appwrite no longer exposes an endpoint for creating project API keys, so this
# resource cannot create one. Create the key in the Appwrite Console, import it,
# and Terraform then manages its name, scopes and expiry from here on.
#
#   terraform import appwrite_project_key.deployments organization-id/project-id/deployments
resource "appwrite_project_key" "deployments" {
  id              = "deployments"
  project_id      = "project-id"
  organization_id = "organization-id"
  name            = "Deployment automation"
  scopes          = ["functions.read", "functions.write", "sites.read", "sites.write"]
}

# The secret is only returned at creation time, so it is empty on an imported
# key. Read it from the Console instead of from state.
output "deployment_api_key" {
  value     = appwrite_project_key.deployments.secret
  sensitive = true
}
