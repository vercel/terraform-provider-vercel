resource "vercel_project" "example" {
  name = "example-project"
}

resource "vercel_project" "example2" {
  name = "another-example-project"
}

resource "vercel_webhook" "with_project_ids" {
  events      = ["deployment.created", "deployment.succeeded"]
  endpoint    = "https://example.com/endpoint"
  project_ids = [vercel_project.example.id, vercel_project.example2.id]
}

resource "vercel_webhook" "without_project_ids" {
  events   = ["deployment.created", "deployment.succeeded"]
  endpoint = "https://example.com/endpoint"
}
