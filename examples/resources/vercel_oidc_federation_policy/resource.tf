resource "vercel_oidc_federation_policy" "turborepo_github_actions" {
  name       = "Turborepo for acme/widgets GitHub Workflows"
  client     = "turborepo"
  issuer_url = "https://token.actions.githubusercontent.com"

  claims = [
    {
      name = "aud"
      values = [
        {
          value = "https://vercel.com/api/login/oauth/token"
        }
      ]
    },
    {
      name = "repository"
      values = [
        {
          value = "acme/widgets"
        }
      ]
    }
  ]

  permissions = ["read-write:remote-cache"]

  resources = {
    project_ids = ["*"]
  }
}
