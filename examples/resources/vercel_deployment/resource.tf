# In this example, we are assuming that a nextjs UI
# exists in a `ui` directory and any terraform exists in a `terraform` directory.
# E.g.
# ```
# ui/
#    src/
#        index.js
#    package.json
#    // etc...
# terraform/
#    main.tf
# ```

data "vercel_project_directory" "example" {
  path = "../ui"
}

data "vercel_project" "example" {
  name = "my-awesome-project"
}

resource "vercel_deployment" "example" {
  project_id  = data.vercel_project.example.id
  files       = data.vercel_project_directory.example.files
  path_prefix = data.vercel_project_directory.example.path
  production  = true

  environment = {
    FOO = "bar"
  }
}
