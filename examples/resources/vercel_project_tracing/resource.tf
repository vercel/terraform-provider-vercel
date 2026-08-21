resource "vercel_project_tracing" "example" {
  project_id = vercel_project.example.id
  enabled    = true

  sampling_rules = [{
    rate         = 0.2
    environment  = "production"
    request_path = "/api"
  }]
}
