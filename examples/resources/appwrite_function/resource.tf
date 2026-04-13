resource "appwrite_function" "hello_world" {
  name       = "hello-world"
  runtime    = "node-22"
  entrypoint = "index.js"
  commands   = "npm install"
}

resource "appwrite_function" "scheduled" {
  name     = "daily-cleanup"
  runtime  = "node-22"
  schedule = "0 0 * * *"
  timeout  = 60
}

resource "appwrite_function" "event_driven" {
  name       = "on-user-create"
  runtime    = "node-22"
  events     = ["users.*.create"]
  entrypoint = "index.js"
  execute    = ["any"]
}
