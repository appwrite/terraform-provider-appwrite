# A server-side project: clients never sign in or upload, so the services
# they would call are switched off. Services left out are not managed.
resource "appwrite_project_services" "example" {
  account   = false
  storage   = false
  messaging = false

  # Explicitly kept on. Removing a line later enables the service again.
  tablesdb  = true
  functions = true
}
