# A project that is connected to a git repository.
# Deployments will be created automatically
# on every branch push and merges onto the Production Branch.
resource "vercel_project" "with_git" {
  name      = "example-project-with-git"
  framework = "nextjs"

  git_repository = {
    type = "github"
    repo = "vercel/some-repo"
  }
}

# A project that is not connected to a git repository.
# Deployments will need to be created manually through
# terraform, or via the vercel CLI.
resource "vercel_project" "example" {
  name                 = "example-project"
  framework            = "nextjs"
  protected_sourcemaps = true
}

locals {
  github_actions_trusted_source = {
    issuer = "https://token.actions.githubusercontent.com"
    label  = "GitHub Actions"
    to = {
      slugs = ["preview"]
    }
    claims = {
      aud = ["example-audience"]
      sub = ["repo:vercel/some-repo:ref:refs/heads/main"]
    }
  }
}

# A project that allows trusted sources to bypass Deployment Protection.
resource "vercel_project" "with_trusted_sources" {
  name      = "example-project-with-trusted-sources"
  framework = "nextjs"

  trusted_sources = {
    projects = [
      {
        project_id = vercel_project.with_git.id
        label      = "Source project"
        custom_allow = [
          {
            from = {
              slugs = ["production"]
            }
            to = {
              slugs = ["preview", "production"]
            }
          }
        ]
      }
    ]

    external_sources = [
      local.github_actions_trusted_source,
    ]
  }
}
