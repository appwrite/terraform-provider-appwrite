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

# Deploy a function from a code upload
resource "appwrite_function_deployment" "process_order_code" {
  function_id = appwrite_function.process_order.id
  source_type = "code"
  code_path   = "./process-order.tar.gz"
  code_hash   = filesha256("./files/process-order.tar.gz")
  entrypoint  = "index.js"
  commands    = "npm install"
  activate    = true
}

# Deploy a function from VCS (requires the function to have a VCS provider configured
# via installation_id, provider_repository_id, and provider_branch)
# resource "appwrite_function_deployment" "daily_report_vcs" {
#   function_id = appwrite_function.daily_report.id
#   source_type = "vcs"
#   type        = "branch"
#   reference   = "main"
#   activate    = true
# }

# Deploy a function from a template
resource "appwrite_function_deployment" "daily_report_template" {
  function_id    = appwrite_function.daily_report.id
  source_type    = "template"
  repository     = "templates"
  owner          = "appwrite"
  root_directory = "node/starter"
  type           = "branch"
  reference      = "main"
  activate       = true
}
