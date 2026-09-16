resource "vercel_project_deployment_check" "e2e" {
  project_id = vercel_project.example.id
  name       = "End-to-end tests"
  requires   = "deployment-url"
  blocks     = "deployment-alias"
  targets    = ["production"]

  source = {
    kind                = "git-provider"
    provider            = "github"
    external_check_name = "e2e"
  }
}
