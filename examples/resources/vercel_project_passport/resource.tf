# Standalone: the project already exists; only Passport is managed here.
resource "vercel_project_passport" "existing" {
  team_id         = "team_example"
  project_id      = "prj_existing"
  connector_id    = "scl_existing"
  deployment_type = "all"
}

# If you ALSO manage this project with vercel_project, set
# manage_passport = false there and omit its inline passport attribute.
# Removing the standalone resource disables Passport, never the project.
