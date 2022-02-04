# A project that is connected to a git repository.
# Deployments will be created automatically
# on every branch push and merges onto the Production Branch.
resource "vercel_project" "with_git" {
  name           = "example-project-with-git"
  framework      = "create-react-app"
  root_directory = "ui"

  environment = [
    {
      key    = "bar"
      value  = "baz"
      target = ["preview"]
    }
  ]

  git_repository = {
    type = "github"
    repo = "vercel/some-repo"
  }
}

# A project that is not connected to a git repository.
# Deployments will need to be created manually through
# terraform, or via the vercel CLI.
resource "vercel_project" "example" {
  name           = "example-project"
  framework      = "create-react-app"
  root_directory = "packages/ui"

  environment = [
    {
      key    = "bar"
      value  = "baz"
      target = ["preview"]
    }
  ]
}
