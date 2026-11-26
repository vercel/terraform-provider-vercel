resource "vercel_project" "example" {
  name = "example-project"
}

resource "vercel_project_speed_insights" "example" {
  project_id = vercel_project.example.id
}
