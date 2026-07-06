resource "vercel_project" "example" {
  name = "example-project-with-vcr-repository"
}

resource "vercel_vcr_repository" "example" {
  project_id = vercel_project.example.id
  name       = "my-repository"
}
