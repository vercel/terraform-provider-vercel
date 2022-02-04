data "vercel_project_directory" "example" {
  path = "../ui"
}

data "vercel_project" "example" {
  name = "my-project"
}

resource "vercel_deployment" "example" {
  project_id = data.vercel_project.example.id
  files      = data.vercel_project_directory.example.files
}
