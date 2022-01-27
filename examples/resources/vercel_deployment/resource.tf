data "vercel_project_directory" "example" {
  path = "packages/ui"
}

resource "vercel_deployment" "example" {
  project_id = "prj_xxxxxxxxxxxx"
  files      = data.vercel_project_directory.example.files
  production = true

  project_settings = {
    output_directory = "packages/ui/.build"
    build_command    = "npm run build && npm run post-build"
    framework        = "create-react-app"
    root_directory   = "packages/ui"
  }

  environment = {
    FOO = "bar"
  }
}
