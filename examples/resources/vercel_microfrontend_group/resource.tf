data "vercel_project" "parent_mfe_project" {
  name = "my parent project"
}

data "vercel_project" "child_mfe_project" {
  name = "my child project"
}

resource "vercel_microfrontend_group" "example_mfe_group" {
  name = "my mfe"
  default_app = {
    project_id = data.vercel_project.parent_mfe_project.id
  }
}

resource "vercel_microfrontend_group_membership" "child_mfe_project_mfe_membership" {
  project_id             = vercel_project.child_mfe_project.id
  microfrontend_group_id = vercel_microfrontend_group.example_mfe_group.id
}
