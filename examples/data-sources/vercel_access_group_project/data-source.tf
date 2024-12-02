data "vercel_project" "example" {
  name = "my-existing-project"
}

data "vercel_access_group_project" "example" {
  access_group_id = "ag_xxxxxxxxxxxxxxxxxxxxxxxxxxxx"
  project_id      = vercel_project.example.id
}
