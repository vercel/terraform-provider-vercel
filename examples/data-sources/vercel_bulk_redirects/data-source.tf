resource "vercel_project" "example" {
  name = "example-project"
}

data "vercel_bulk_redirects" "example" {
  project_id = vercel_project.example.id
}
