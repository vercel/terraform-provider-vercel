resource "vercel_project" "example" {
  name = "example"

  git_repository = {
    type = "github"
    repo = "vercel/some-repo"
  }
}

# An environment variable that will be created
# and associated with the "example" project.
resource "vercel_shared_environment_variable" "example" {
  key    = "EXAMPLE"
  value  = "some_value"
  target = ["production"]
  project_ids = [
    vercel_project.example.id
  ]
}
