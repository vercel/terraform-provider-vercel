data "vercel_file" "example" {
  path = "index.html"
}

data "vercel_project" "example" {
  name = "my-project"
}

resource "vercel_deployment" "example" {
  project_id = data.vercel_project.example.id
  files      = data.vercel_file.example.file
}
