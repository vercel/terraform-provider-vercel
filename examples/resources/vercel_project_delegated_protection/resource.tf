resource "vercel_project" "example" {
  name = "example-project"

  git_repository = {
    type = "github"
    repo = "vercel/some-repo"
  }
}

resource "vercel_project_delegated_protection" "example" {
  project_id      = vercel_project.example.id
  client_id       = "client-id"
  client_secret   = "client-secret"
  deployment_type = "standard_protection_new"
  issuer          = "https://auth.example.com"
}
