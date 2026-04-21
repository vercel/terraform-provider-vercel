resource "vercel_project" "example" {
  name = "example-project"

  git_repository = {
    type = "github"
    repo = "vercel/some-repo"
  }
}

# Project environment variables must explicitly set `sensitive`.
resource "vercel_project_environment_variable" "example" {
  project_id = vercel_project.example.id
  key        = "foo"
  value      = "bar"
  target     = ["production"]
  sensitive  = true
  comment    = "a production secret"
}

# An environment variable that will be created
# for this project for the "preview" environment when the branch is "staging".
resource "vercel_project_environment_variable" "example_git_branch" {
  project_id = vercel_project.example.id
  key        = "foo"
  value      = "bar-staging"
  target     = ["preview"]
  sensitive  = true
  git_branch = "staging"
  comment    = "a staging secret"
}

# Development environment variables must explicitly set `sensitive = false`.
resource "vercel_project_environment_variable" "example_development" {
  project_id = vercel_project.example.id
  key        = "foo-development"
  value      = "bar-development"
  target     = ["development"]
  sensitive  = false
  comment    = "available during local development"
}

# An environment variable that will be created referencing
# an ephemeral source whose values won't save to state.
ephemeral "vault_kv_secret_v2" "example" {
  mount = "kv"
  name  = "example"
}
resource "vercel_project_environment_variable" "example_ephemeral" {
  project_id = vercel_project.example.id
  key        = "foo"
  value_wo   = ephemeral.vault_kv_secret_v2.example.data["example"]
  target     = ["production"]
  sensitive  = true
  comment    = "an ephemeral secret"
}
