resource "vercel_project" "example" {
  name = "example-project"

  git_repository = {
    type = "github"
    repo = "vercel/some-repo"
  }
}

# An environment variable that will be created
# for this project for the "production" environment.
resource "vercel_project_environment_variable" "example" {
  project_id = vercel_project.example.id
  key        = "foo"
  value      = "bar"
  target     = ["production"]
  comment    = "a production secret"
}

# An environment variable that will be created
# for this project for the "preview" environment when the branch is "staging".
resource "vercel_project_environment_variable" "example_git_branch" {
  project_id = vercel_project.example.id
  key        = "foo"
  value      = "bar-staging"
  target     = ["preview"]
  git_branch = "staging"
  comment    = "a staging secret"
}

# A sensitive environment variable that will be created
# for this project for the "production" environment.
resource "vercel_project_environment_variable" "example_sensitive" {
  project_id = vercel_project.example.id
  key        = "foo"
  value      = "bar-production"
  target     = ["production"]
  sensitive  = true
  comment    = "a sensitive production secret"
}

# An environment variable that will be created referencing
# an ephemeral source whose value won't save to state.
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
