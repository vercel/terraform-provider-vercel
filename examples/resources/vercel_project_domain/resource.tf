resource "vercel_project" "example" {
  name      = "example_project"
  framework = "create-react-app"
}

resource "vercel_project_domain" "example" {
  project_id = vercel_project.example.id
  domain     = "i-love.vercel.app"
}

resource "vercel_project_domain" "example_redirect" {
  project_id = vercel_project.example.id
  domain     = "i-also-love.vercel.app"

  redirect             = vercel_project_domain.example.domain
  redirect_status_code = 307
}
