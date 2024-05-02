resource "vercel_project" "example" {
  name = "example-project"
}

resource "vercel_attack_challenge_mode" "example" {
  project_id = vercel_project.example.id
  enabled    = true
}
