data "vercel_project_directory" "example" {
  path = "../ui"
}

data "vercel_project" "example" {
  name = "my-awesome-project"
}

resource "vercel_deployment" "example" {
  project_id = data.vercel_project.example.id
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
