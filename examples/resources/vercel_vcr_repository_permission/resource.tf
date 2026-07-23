resource "vercel_project" "example" {
  name = "example-project-with-vcr-repository"
}

resource "vercel_vcr_repository" "example" {
  project_id = vercel_project.example.id
  name       = "my-repository"
}

# Share the repository with another team, granting it
# read (pull) access to the repository's images.
resource "vercel_vcr_repository_permission" "example" {
  project_id      = vercel_project.example.id
  repository      = vercel_vcr_repository.example.name
  granted_team_id = "team_xxxxxxxxxxxxxxxxxxxxxxxx"
}

# The granted team can alternatively be referenced by its slug.
resource "vercel_vcr_repository_permission" "example_by_slug" {
  project_id        = vercel_project.example.id
  repository        = vercel_vcr_repository.example.name
  granted_team_slug = "my-other-team"
}
