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

resource "vercel_oidc_federation_policy" "vcr_github_actions" {
  name       = "Push acme/widgets images from its main branch"
  client     = "vcr"
  issuer_url = "https://token.actions.githubusercontent.com"

  claims = [
    {
      name = "aud"
      values = [
        {
          value = "https://github.com/acme"
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
    },
    {
      name = "ref"
      values = [
        {
          value = "refs/heads/main"
        }
      ]
    },
    {
      name = "workflow_ref"
      values = [
        {
          value = "acme/widgets/.github/workflows/image.yml@refs/heads/main"
        }
      ]
    }
  ]

  permissions = ["read-write:vcr"]

  resources = {
    project_ids = ["prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx"]
  }
}
