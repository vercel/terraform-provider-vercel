variable "integration_token" {
  description = "An integration installation token authorized for this project."
  type        = string
  sensitive   = true
}

variable "project_id" {
  description = "The ID or name of an existing project accessible to the integration."
  type        = string
}

variable "team_id" {
  description = "The team that owns the project."
  type        = string
}

provider "vercel" {
  alias     = "integration"
  api_token = var.integration_token
  team      = var.team_id
}

resource "vercel_project_deployment_check" "integration" {
  provider   = vercel.integration
  project_id = var.project_id
  name       = "Integration quality gate"
  requires   = "build-ready"
  blocks     = "deployment-alias"
  targets    = ["production"]

  source = {
    kind = "integration"
    # Optionally associate a resource belonging to this integration installation:
    # external_resource_id = "external-resource-id"
  }
}
