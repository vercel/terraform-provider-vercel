data "vercel_shared_environment_variable" "example" {
  key    = "EXAMPLE_ENV_VAR"
  target = ["production", "preview"]
}

resource "vercel_project" "example" {
  name = "example"
}

resource "vercel_shared_environment_variable_project_link" "example" {
  shared_environment_variable_id = data.vercel_shared_environment_variable.example.id
  project_id                     = vercel_project.example.id
}
