data "vercel_project" "example" {
  name = "my-existing-project"
}

data "vercel_project_secure_compute_networks" "example" {
  project_id = data.vercel_project.example.id
}
