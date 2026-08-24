resource "vercel_project" "example" {
  name = "example-project"
}

resource "vercel_kms_issuer" "example" {
  name = "my-issuer"
}

resource "vercel_kms_project_grant" "example" {
  issuer_id    = vercel_kms_issuer.example.id
  project_id   = vercel_project.example.id
  environments = ["production", "preview"]

  token_claims = jsonencode({
    aud = "https://example.com"
  })
}
