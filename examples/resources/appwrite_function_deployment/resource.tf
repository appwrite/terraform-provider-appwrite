resource "appwrite_function_deployment" "from_code" {
  function_id = appwrite_function.example.id
  source_type = "code"
  code_path   = "dist/function.tar.gz"
  code_hash   = filesha256("dist/function.tar.gz")
  entrypoint  = "index.js"
  commands    = "npm install"
  activate    = true
}

resource "appwrite_function_deployment" "from_template" {
  function_id    = appwrite_function.example.id
  source_type    = "template"
  repository     = "starter-template"
  owner          = "appwrite"
  root_directory = "node"
  type           = "branch"
  reference      = "main"
  activate       = true
}
