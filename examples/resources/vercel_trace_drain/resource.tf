resource "vercel_trace_drain" "example" {
  name            = "example-trace-drain"
  delivery_format = "json"
  endpoint        = "https://example.com/v1/traces"

  project_ids = [vercel_project.example.id]

  sampling_rules = [{
    rate         = 0.8
    environment  = "production"
    request_path = "/api"
  }]
}
