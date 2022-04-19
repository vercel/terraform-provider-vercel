# In this example, we are assuming that a single index.html file
# is being deployed. This file lives directly next to the terraform file.

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
