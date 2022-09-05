# In this example, we are assuming that a nextjs UI exists in a `ui` directory 
# and has been prebuilt via `vercel build`. 
# We assume any terraform code exists in a separate `terraform` directory.
# E.g.
# ```
# ui/
#    .vercel/
#        output/ 
#            ...
#    src/
#        index.js
#    package.json
#    ...
# terraform/
#    main.tf
#    ...
# ```

data "vercel_project" "example" {
  name = "my-awesome-project"
}

data "vercel_prebuilt_project" "example" {
  path = "../ui"
}

resource "vercel_deployment" "example" {
  project_id  = data.vercel_project.example.id
  files       = data.vercel_prebuilt_project.example.output
  path_prefix = data.vercel_prebuilt_project.example.path
}
