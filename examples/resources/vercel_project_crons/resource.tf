resource "vercel_project" "example" {
  name      = "example-project"
  framework = "nextjs"

  git_repository = {
    type = "github"
    repo = "vercel/some-repo"
  }
}

resource "vercel_project_crons" "example" {
  project_id = vercel_project.example.id
  enabled    = true
}
