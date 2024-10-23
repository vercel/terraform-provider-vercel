resource "vercel_project" "example" {
  name = "example"

  git_repository = {
    type = "github"
    repo = "vercel/some-repo"
  }
}

# A shared environment variable that will be created
# and associated with the "example" project.
resource "vercel_shared_environment_variable" "example" {
  key     = "EXAMPLE"
  value   = "some_value"
  target  = ["production"]
  comment = "an example shared variable"
  project_ids = [
    vercel_project.example.id
  ]
}
