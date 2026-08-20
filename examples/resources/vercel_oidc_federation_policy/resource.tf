resource "vercel_oidc_federation_policy" "turborepo_github_actions" {
  name       = "Turborepo for acme/widgets GitHub Workflows"
  client_id  = "cl_kyUx2zVvA4MGptBohkmtYHJly2XltXzD"
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

  permissions = ["*"]

  resources = {
    project_ids = ["*"]
  }
}
