resource "vercel_project" "example" {
  name      = "deployment-protection-exception-example"
  framework = "nextjs"
}

resource "vercel_deployment_protection_exception" "example" {
  project_id = vercel_project.example.id
  alias      = "preview-branch-name.vercel.app"
}
