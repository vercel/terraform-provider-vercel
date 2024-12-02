resource "vercel_project" "example" {
  name = "example-project"
}

resource "vercel_access_group" "example" {
  name = "example-access-group"
}

resource "vercel_access_group_project" "example" {
  project_id      = vercel_project.example.id
  access_group_id = vercel_access_group.example.id
  role            = "ADMIN"
}
