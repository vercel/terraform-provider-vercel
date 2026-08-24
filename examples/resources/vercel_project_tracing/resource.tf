resource "vercel_project_tracing" "example" {
  project_id = vercel_project.example.id

  sampling_rules = [{
    rate         = 0.2
    environment  = "production"
    request_path = "/api"
  }]
}
