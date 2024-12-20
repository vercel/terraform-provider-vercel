resource "vercel_project" "example" {
  name = "example-project"

  git_repository = {
    type = "github"
    repo = "vercel/some-repo"
  }
}

resource "vercel_project_environment_variables" "example" {
  project_id = vercel_project.test.id
  upsert     = true
  variables = [
    {
      key    = "SOME_VARIABLE"
      value  = "some_value"
      target = ["production", "preview"]
    },
    {
      key        = "ANOTHER_VARIABLE"
      value      = "another_value"
      git_branch = "staging"
      target     = ["preview"]
    },
    {
      key       = "SENSITIVE_VARIABLE"
      value     = "sensitive_value"
      target    = ["production"]
      sensitive = true
    }
  ]
}
