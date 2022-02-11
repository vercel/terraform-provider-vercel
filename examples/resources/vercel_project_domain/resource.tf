resource "vercel_project" "example" {
  name = "example-project"
}

# A simple domain that will be automatically
# applied to each production deployment
resource "vercel_project_domain" "example" {
  project_id = vercel_project.example.id
  domain     = "i-love.vercel.app"
}

# A redirect of a domain name to a second domain name.
# The status_code can optionally be controlled.
resource "vercel_project_domain" "example_redirect" {
  project_id = vercel_project.example.id
  domain     = "i-also-love.vercel.app"

  redirect             = vercel_project_domain.example.domain
  redirect_status_code = 307
}
