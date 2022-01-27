data "vercel_file" "example" {
  path = "index.html"
}

resource "vercel_deployment" "example" {
  project_id = "prj_xxxxxxxxxxxx"
  files      = data.vercel_file.example.file
}
