variable "github_repository" {
  description = "GitHub repository connected to Vercel, in owner/repository format."
  type        = string
}

resource "vercel_project" "example" {
  name = "deployment-checks-example"

  git_repository = {
    type = "github"
    repo = var.github_repository
  }
}

# The external check name must match the check reported by your GitHub workflow.
# This resource registers a gate; it does not create or run the workflow.
resource "vercel_project_deployment_check" "e2e" {
  project_id       = vercel_project.example.id
  name             = "End-to-end tests"
  requires         = "deployment-url"
  blocks           = "deployment-alias"
  targets          = ["production"]
  is_rerequestable = false

  source = {
    kind                = "git-provider"
    provider            = "github"
    external_check_name = "e2e"
  }
}

# A webhook-backed check for an external test runner. The runner must handle
# check events and report results through the Checks API separately.
resource "vercel_project_deployment_check" "webhook" {
  project_id       = vercel_project.example.id
  name             = "External smoke tests"
  requires         = "deployment-url"
  blocks           = "none"
  targets          = ["preview"]
  is_rerequestable = true

  source = {
    kind = "webhook"
    # Optionally associate an existing webhook:
    # webhook_id = "hook_xxx"
  }
}
