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

data "vercel_project_directory" "files_example" {
  path = "../ui"
}

data "vercel_project" "files_example" {
  name = "my-awesome-project"
}

resource "vercel_deployment" "files_example" {
  project_id  = data.vercel_project.files_example.id
  files       = data.vercel_project_directory.files_example.files
  path_prefix = data.vercel_project_directory.files_example.path
  production  = true

  environment = {
    FOO = "bar"
  }
}

## Or deploying a specific commit or branch
resource "vercel_project" "git_example" {
  name      = "my-awesome-git-project"
  framework = "nextjs"
  git_repository = {
    type = "github"
    repo = "vercel/some-repo"
  }
}

resource "vercel_deployment" "git_example" {
  project_id = vercel_project.git_example.id
  ref        = "d92f10e" # or a git branch
}
