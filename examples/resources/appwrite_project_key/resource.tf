resource "appwrite_project_key" "deployments" {
  id              = "deployments"
  project_id      = "project-id"
  organization_id = "organization-id"
  name            = "Deployment automation"
  scopes          = ["functions.read", "functions.write", "sites.read", "sites.write"]
}

output "deployment_api_key" {
  value     = appwrite_project_key.deployments.secret
  sensitive = true
}
