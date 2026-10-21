resource "vercel_project_deployment_check" "vercel_ci" {
  project_id = vercel_project.example.id
  name       = "E2E tests"
  requires   = "none"
  blocks     = "deployment-alias"
  targets    = ["production"]

  source = {
    kind     = "vercel"
    sub_kind = "vercel-ci-check"

    selection = {
      kind             = "turborepo"
      task             = "e2e:ci"
      filters          = ["vercel-docs"]
      fail_if_no_match = true
    }
  }
}
