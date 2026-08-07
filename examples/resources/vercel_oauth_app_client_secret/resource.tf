resource "vercel_oauth_app" "example" {
  name = "My Example App"
  slug = "my-example-app"

  redirect_uris = ["https://example.com/api/auth/callback"]
  scopes        = ["openid", "email", "profile", "offline_access"]
}

resource "vercel_oauth_app_client_secret" "example" {
  oauth_app_id = vercel_oauth_app.example.id

  # An OAuth App can hold at most two secrets at a time, so a replacement
  # secret can be created before the old one is destroyed (zero-downtime
  # rotation with e.g. `terraform apply -replace=vercel_oauth_app_client_secret.example`).
  lifecycle {
    create_before_destroy = true
  }
}

# Example: feed the credentials to the application consuming them.
resource "vercel_project_environment_variable" "oauth_client_id" {
  project_id = vercel_project.example.id
  key        = "OAUTH_CLIENT_ID"
  value      = vercel_oauth_app.example.id
  target     = ["production", "preview", "development"]
}

resource "vercel_project_environment_variable" "oauth_client_secret" {
  project_id = vercel_project.example.id
  key        = "OAUTH_CLIENT_SECRET"
  value      = vercel_oauth_app_client_secret.example.client_secret
  target     = ["production", "preview", "development"]
  sensitive  = true
}

resource "vercel_project" "example" {
  name = "example-project"
}
