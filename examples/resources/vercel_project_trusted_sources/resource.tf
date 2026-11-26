# Standalone: this manages Trusted Sources on a project that already exists.
# It does not create or own the project itself.
resource "vercel_project_trusted_sources" "existing" {
  team_id    = "team_example"
  project_id = "prj_existing"

  # Other Vercel projects in the same team that may call this project's
  # protected deployments.
  projects = [
    {
      project_id = "prj_frontend"
      label      = "Frontend"
    },
  ]

  # External OIDC sources, such as GitHub Actions.
  external_sources = [
    {
      issuer = "https://token.actions.githubusercontent.com"
      label  = "GitHub Actions"
      to     = { slugs = ["preview"] }
      claims = {
        aud        = ["https://vercel.com/example"]
        repository = ["example/app"]
      }
    },
  ]

  enable_vercel_ci_same_repository = true
}

# If vercel_project also manages this project, set
# manage_trusted_sources = false there and omit its inline trusted_sources.
