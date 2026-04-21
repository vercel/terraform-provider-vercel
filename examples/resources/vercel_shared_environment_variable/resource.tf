resource "vercel_project" "example" {
  name = "example"

  git_repository = {
    type = "github"
    repo = "vercel/some-repo"
  }
}

# Shared environment variables must explicitly set `sensitive`.
resource "vercel_shared_environment_variable" "example" {
  key     = "EXAMPLE"
  value   = "some_value"
  target  = ["production"]
  sensitive = true
  comment = "an example shared variable"
  project_ids = [
    vercel_project.example.id
  ]
}

# Shared environment variables targeting `development` must explicitly set `sensitive = false`.
resource "vercel_shared_environment_variable" "example_development" {
  key       = "EXAMPLE_DEVELOPMENT"
  value     = "some_development_value"
  target    = ["development"]
  sensitive = false
  comment   = "available during local development"
  project_ids = [
    vercel_project.example.id
  ]
}
