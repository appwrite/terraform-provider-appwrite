# Serve the project over REST only. Protocols left out are not managed.
resource "appwrite_project_protocols" "example" {
  graphql   = false
  websocket = false
}
