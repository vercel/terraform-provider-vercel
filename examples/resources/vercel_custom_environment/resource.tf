resource "vercel_project" "example" {
  name = "example-project-with-custom-env"
}

resource "vercel_custom_environment" "example" {
  project_id  = vercel_project.example.id
  name        = "example-custom-env"
  description = "A description of the custom environment"
  branch_tracking = {
    pattern = "staging-"
    type    = "startsWith"
  }
}
