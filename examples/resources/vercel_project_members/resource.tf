resource "vercel_project" "example" {
  name = "example-with-members"
}

resource "vercel_project_members" "example" {
  project_id = vercel_project.example.id

  members = [{
    email = "user@example.com"
    role  = "PROJECT_VIEWER"
    }, {
    username = "some-example-user"
    role     = "PROJECT_DEVELOPER"
  }]
}
