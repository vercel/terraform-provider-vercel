resource "vercel_project" "example" {
  name = "example-project"

  git_repository = {
    type = "github"
    repo = "vercel/some-repo"
  }
}

resource "vercel_project_environment_variables" "example" {
  project_id = vercel_project.example.id
  variables = [
    {
      key    = "SOME_VARIABLE"
      value  = "some_value"
      target = ["production", "preview"]
      sensitive = true
    },
    {
      key        = "ANOTHER_VARIABLE"
      value      = "another_value"
      git_branch = "staging"
      target     = ["preview"]
      sensitive  = true
    },
    {
      key       = "DEVELOPMENT_VARIABLE"
      value     = "development_value"
      target    = ["development"]
      sensitive = false
    }
  ]
}
