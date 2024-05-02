data "vercel_project" "example" {
  name = "example"
}

data "vercel_project_function_cpu" "example" {
  project_id = data.vercel_project.example.id
}
