resource "appwrite_function_variable" "api_url" {
  function_id = appwrite_function.hello_world.id
  key         = "API_URL"
  value       = "https://api.example.com"
}

resource "appwrite_function_variable" "secret_key" {
  function_id = appwrite_function.hello_world.id
  key         = "SECRET_KEY"
  value       = var.secret_key
  secret      = true
}
