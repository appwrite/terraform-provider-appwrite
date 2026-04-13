resource "appwrite_site" "landing_page" {
  name          = "landing-page"
  framework     = "other"
  build_runtime = "node-22"
}

resource "appwrite_site" "dashboard" {
  name            = "dashboard"
  framework       = "nextjs"
  build_runtime   = "node-22"
  install_command = "npm install"
  build_command   = "npm run build"
}

resource "appwrite_site" "docs" {
  name             = "docs"
  framework        = "astro"
  build_runtime    = "node-22"
  install_command  = "npm install"
  build_command    = "npm run build"
  output_directory = "dist"
}
