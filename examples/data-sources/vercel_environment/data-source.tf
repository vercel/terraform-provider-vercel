data "vercel_project" "example" {
  name = "example-project"
}

data "vercel_environment" "preview" {
  project_id = data.vercel_project.example.id
  slug       = "preview"
}

data "vercel_environment" "staging" {
  project_id = data.vercel_project.example.id
  id         = "env_xxxxxxxxxxxxxxxxxxxxxxxxxxxx"
}
