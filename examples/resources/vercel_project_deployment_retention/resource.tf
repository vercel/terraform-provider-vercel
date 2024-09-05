resource "vercel_project" "example" {
  name = "example-project"

  git_repository = {
    type = "github"
    repo = "vercel/some-repo"
  }
}

# An unlimited deployment retention policy that will be created
# for this project for all deployments.
resource "vercel_project_deployment_retention" "example_unlimited" {
  project_id            = vercel_project.example.id
  team_id               = vercel_project.example.team_id
  expiration_preview    = "unlimited"
  expiration_production = "unlimited"
  expiration_canceled   = "unlimited"
  expiration_errored    = "unlimited"
}

# A customized deployment retention policy that will be created
# for this project for all deployments.
resource "vercel_project_deployment_retention" "example_customized" {
  project_id            = vercel_project.example.id
  team_id               = vercel_project.example.team_id
  expiration_preview    = "3m"
  expiration_production = "1y"
  expiration_canceled   = "1m"
  expiration_errored    = "2m"
}
