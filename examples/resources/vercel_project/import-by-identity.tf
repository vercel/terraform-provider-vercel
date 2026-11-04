# Import adopts the whole project. Destroying the resource deletes the project.
# Use lifecycle.prevent_destroy if Terraform should reject project deletion.
import {
  to = vercel_project.example
  identity = {
    project_id = "prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx"
    # Omit team_id to use the default team configured on the provider.
    team_id = "team_xxxxxxxxxxxxxxxxxxxxxxxx"
  }
}
