resource "appwrite_site_deployment" "from_code" {
  site_id     = appwrite_site.example.id
  source_type = "code"
  code_path   = "dist/site.tar.gz"
  code_hash   = filesha256("dist/site.tar.gz")
  activate    = true
}

resource "appwrite_site_deployment" "from_template" {
  site_id        = appwrite_site.example.id
  source_type    = "template"
  repository     = "starter-template"
  owner          = "appwrite"
  root_directory = "nextjs"
  type           = "branch"
  reference      = "main"
  activate       = true
}
