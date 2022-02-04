data "vercel_project" "foo" {
  name = "my-existing-project"
}

# Outputs prj_xxxxxx
output "project_id" {
  value = data.vercel_project.foo.id
}
