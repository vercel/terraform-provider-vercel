---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "vercel_custom_environment Data Source - terraform-provider-vercel"
subcategory: ""
description: |-
  Provides information about an existing CustomEnvironment resource.
  An CustomEnvironment allows a vercel_deployment to be accessed through a different URL.
---

# vercel_custom_environment (Data Source)

Provides information about an existing CustomEnvironment resource.

An CustomEnvironment allows a `vercel_deployment` to be accessed through a different URL.

## Example Usage

```terraform
data "vercel_project" "example" {
  name = "example-project-with-custom-env"
}

data "vercel_custom_environment" "example" {
  project_id = data.vercel_project.example.id
  name       = "example-custom-env"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The name of the environment.
- `project_id` (String) The ID of the existing Vercel Project.

### Optional

- `team_id` (String) The team ID to add the project to. Required when configuring a team resource if a default team has not been set in the provider.

### Read-Only

- `branch_tracking` (Attributes) The branch tracking configuration for the environment. When enabled, each qualifying merge will generate a deployment. (see [below for nested schema](#nestedatt--branch_tracking))
- `description` (String) A description of what the environment is.
- `id` (String) The ID of the environment.

<a id="nestedatt--branch_tracking"></a>
### Nested Schema for `branch_tracking`

Read-Only:

- `pattern` (String) The pattern of the branch name to track.
- `type` (String) How a branch name should be matched against the pattern. Must be one of 'startsWith', 'endsWith' or 'equals'.
