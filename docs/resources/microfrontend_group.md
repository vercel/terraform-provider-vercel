---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "vercel_microfrontend_group Resource - terraform-provider-vercel"
subcategory: ""
description: |-
  Provides a Microfrontend Group resource.
  A Microfrontend Group is a definition of a microfrontend belonging to a Vercel Team.
---

# vercel_microfrontend_group (Resource)

Provides a Microfrontend Group resource.

A Microfrontend Group is a definition of a microfrontend belonging to a Vercel Team.

## Example Usage

```terraform
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
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `default_app` (Attributes) The default app for the project. Used as the entry point for the microfrontend. (see [below for nested schema](#nestedatt--default_app))
- `name` (String) A human readable name for the microfrontends group.

### Optional

- `team_id` (String) The team ID to add the microfrontend group to. Required when configuring a team resource if a default team has not been set in the provider.

### Read-Only

- `id` (String) A unique identifier for the group of microfrontends. Example: mfe_12HKQaOmR5t5Uy6vdcQsNIiZgHGB
- `slug` (String) A slugified version of the name.

<a id="nestedatt--default_app"></a>
### Nested Schema for `default_app`

Required:

- `project_id` (String) The ID of the project.

Optional:

- `default_route` (String) The default route for the project. Used for the screenshot of deployments.

## Import

Import is supported using the following syntax:

```shell
# If importing into a personal account, or with a team configured on the provider, simply use the record id.
# - the microfrontend ID can be taken from the microfrontend settings page
terraform import vercel_microfrontend_group.example mfe_xxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Alternatively, you can import via the team_id and microfrontend_id.
# - team_id can be found in the team `settings` tab in the Vercel UI.
# - the microfrontend ID can be taken from the microfrontend settings page
terraform import vercel_microfrontend_group.example team_xxxxxxxxxxxxxxxxxxxxxxxx/mfe_xxxxxxxxxxxxxxxxxxxxxxxxxxxx
```
