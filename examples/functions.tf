resource "appwrite_function" "process_order" {
  name       = "process-order"
  runtime    = "node-22"
  entrypoint = "index.js"
  commands   = "npm install"
  events     = ["databases.*.collections.*.documents.*.create"]
  timeout    = 30
}

resource "appwrite_function" "daily_report" {
  name       = "daily-report"
  runtime    = "node-22"
  entrypoint = "index.js"
  commands   = "npm install"
  schedule   = "0 9 * * *"
  timeout    = 120
}

resource "appwrite_function_variable" "process_order_api_url" {
  function_id = appwrite_function.process_order.id
  key         = "API_URL"
  value       = "https://api.example.com"
}

resource "appwrite_function_variable" "daily_report_smtp_host" {
  function_id = appwrite_function.daily_report.id
  key         = "SMTP_HOST"
  value       = "smtp.example.com"
}
