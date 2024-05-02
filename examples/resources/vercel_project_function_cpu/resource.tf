resource "vercel_project" "example" {
  name = "example"
}

resource "vercel_project_function_cpu" "example" {
  project_id = vercel_project.example.id
  cpu        = "performance"
}
