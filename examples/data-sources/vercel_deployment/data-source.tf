data "vercel_project_directory" "example" {
  path = "packages/ui"
}

resource "vercel_deployment" "example" {
  project_id = "prj_xxxxxxxxxxxx"
  files      = data.vercel_project_directory.example.files
}
