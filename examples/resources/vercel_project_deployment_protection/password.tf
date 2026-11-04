terraform {
  required_version = ">= 1.11.0"
}

variable "deployment_password" {
  type      = string
  sensitive = true
  ephemeral = true
}

# A separate existing project, protected by a password only.
resource "vercel_project_deployment_protection" "password" {
  project_id = "prj_password_project"

  password_protection = {
    deployment_type     = "preview"
    password_wo         = var.deployment_password
    password_wo_version = 1
  }
}
