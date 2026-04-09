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
  name      = "example-project"
  framework = "nextjs"
}

# Back-compatible management of the deployment secret exposed as
# VERCEL_AUTOMATION_BYPASS_SECRET. Additional secrets created outside
# Terraform are preserved.
resource "vercel_project" "legacy_automation_bypass" {
  name      = "example-project-legacy-automation-bypass"
  framework = "nextjs"

  protection_bypass_for_automation        = true
  protection_bypass_for_automation_secret = "12345678912345678912345678912345"
}

# Authoritative management of all automation bypass secrets on the project.
# Exactly one secret must be selected for VERCEL_AUTOMATION_BYPASS_SECRET.
resource "vercel_project" "managed_automation_bypasses" {
  name      = "example-project-managed-automation-bypasses"
  framework = "nextjs"

  protection_bypass_for_automation = true
  protection_bypass_for_automation_secrets = [
    {
      secret     = "12345678912345678912345678912345"
      note       = "GitHub Actions"
      is_env_var = true
    },
    {
      secret     = "abcdefghijklmnopqrstuvwxyz123456"
      note       = "Smoke tests"
      is_env_var = false
    },
  ]
}
