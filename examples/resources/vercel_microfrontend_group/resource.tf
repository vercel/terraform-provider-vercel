data "vercel_project" "parent-mfe-project" {
  name = "my parent project"
}

data "vercel_project" "child-mfe-project" {
  name = "my child project"
}

resource "vercel_microfrontend_group" "example-mfe-group" {
  name        = "my mfe"
  default_app = vercel_project.parent-mfe-project.id
}

resource "vercel_microfrontend_group_membership" "parent-mfe-project-mfe-membership" {
  project_id             = vercel_project.parent-mfe-project.id
  microfrontend_group_id = vercel_microfrontend_group.example-mfe-group.id
}

resource "vercel_microfrontend_group_membership" "child-mfe-project-mfe-membership" {
  project_id             = vercel_project.child-mfe-project.id
  microfrontend_group_id = vercel_microfrontend_group.example-mfe-group.id
}
