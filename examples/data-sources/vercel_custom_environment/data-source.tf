data "vercel_project" "example" {
  name = "example-project-with-custom-env"
}

data "vercel_custom_environment" "example" {
  project_id = data.vercel_project.example.id
  name       = "example-custom-env"
}
