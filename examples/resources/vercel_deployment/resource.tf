# In this example, we are assuming that a nextjs UI
# exists in a `ui` directory alongside any terraform.
# E.g.
# ```
# ui/
#    src/
#    next.config.js
#    // etc...
# main.tf
# ```

data "vercel_project_directory" "example" {
  path = "ui"
}

data "vercel_project" "example" {
  name = "my-awesome-project"
  # The root directory here is also set to reflect the
  # file structure.
  root_directory = "ui"
}

resource "vercel_deployment" "example" {
  project_id = data.vercel_project.example.id
  files      = data.vercel_project_directory.example.files
  production = true

  environment = {
    FOO = "bar"
  }
}
