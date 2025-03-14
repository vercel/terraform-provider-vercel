---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "vercel_project_domain Resource - terraform-provider-vercel"
subcategory: ""
description: |-
  Provides a Project Domain resource.
  A Project Domain is used to associate a domain name with a vercel_project.
  By default, Project Domains will be automatically applied to any production deployments.
---

# vercel_project_domain (Resource)

Provides a Project Domain resource.

A Project Domain is used to associate a domain name with a `vercel_project`.

By default, Project Domains will be automatically applied to any `production` deployments.

## Example Usage

```terraform
resource "vercel_project" "example" {
  name = "example-project"
}

# A simple domain that will be automatically
# applied to each production deployment
resource "vercel_project_domain" "example" {
  project_id = vercel_project.example.id
  domain     = "i-love.vercel.app"
}

# A redirect of a domain name to a second domain name.
# The status_code can optionally be controlled.
resource "vercel_project_domain" "example_redirect" {
  project_id = vercel_project.example.id
  domain     = "i-also-love.vercel.app"

  redirect             = vercel_project_domain.example.domain
  redirect_status_code = 307
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `domain` (String) The domain name to associate with the project.
- `project_id` (String) The project ID to add the deployment to.

### Optional

- `custom_environment_id` (String) The name of the Custom Environment to link to the Project Domain. Deployments from this custom environment will be assigned the domain name.
- `git_branch` (String) Git branch to link to the project domain. Deployments from this git branch will be assigned the domain name.
- `redirect` (String) The domain name that serves as a target destination for redirects.
- `redirect_status_code` (Number) The HTTP status code to use when serving as a redirect.
- `team_id` (String) The ID of the team the project exists under. Required when configuring a team resource if a default team has not been set in the provider.

### Read-Only

- `id` (String) The ID of this resource.

## Import

Import is supported using the following syntax:

```shell
# If importing into a personal account, or with a team configured on
# the provider, simply use the project ID and domain.
# - project_id can be found in the project `settings` tab in the Vercel UI.
terraform import vercel_project_domain.example prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/example.com

# Alternatively, you can import via the team_id, project_id and domain name.
# - team_id can be found in the team `settings` tab in the Vercel UI.
# - project_id can be found in the project `settings` tab in the Vercel UI.
terraform import vercel_project_domain.example team_xxxxxxxxxxxxxxxxxxxxxxxx/prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/example.com
```
