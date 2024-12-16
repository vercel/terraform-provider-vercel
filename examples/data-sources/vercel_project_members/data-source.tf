data "vercel_project" "example" {
  name = "example-with-members"
}

data "vercel_project_members" "example" {
  project_id = data.vercel_project.example.id
}
