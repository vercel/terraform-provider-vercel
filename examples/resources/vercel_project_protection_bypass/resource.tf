resource "vercel_project" "example" {
  name = "example-project"
}

# A bypass for a CI pipeline. Because this is the first bypass on the project,
# Vercel automatically marks it as the env-var default (is_env_var = true), and
# the secret is exposed as VERCEL_AUTOMATION_BYPASS_SECRET on deployments.
resource "vercel_project_protection_bypass" "ci" {
  project_id = vercel_project.example.id
  note       = "ci pipeline"
}

# A second bypass for QA, with a caller-supplied secret.
resource "vercel_project_protection_bypass" "qa" {
  project_id = vercel_project.example.id
  note       = "preview QA"
  secret     = "abcdefghijklmnopqrstuvwxyz123456"
}
