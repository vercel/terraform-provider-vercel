data "vercel_project_directory" "example" {
  path = "../ui"
}

resource "vercel_deployment" "example" {
  project_id = "prj_xxxxxxxxxxxx"
  files      = data.vercel_project_directory.example.files
  production = true

  project_settings = {
    output_directory = ".build"
    build_command    = "npm run build && npm run post-build"
    framework        = "create-react-app"
    root_directory   = "../ui"
  }

  environment = {
    FOO = "bar"
  }
}
