resource "vercel_project" "example" {
  name = "example"
}

resource "vercel_project_function_max_duration" "example" {
  project_id   = vercel_project.example.id
  max_duration = 100
}
