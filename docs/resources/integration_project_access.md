---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "vercel_integration_project_access Resource - terraform-provider-vercel"
subcategory: ""
description: |-
  Provides Project access to an existing Integration. This requires the integration already exists and is already configured for Specific Project access.
---

# vercel_integration_project_access (Resource)

Provides Project access to an existing Integration. This requires the integration already exists and is already configured for Specific Project access.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `integration_id` (String) The ID of the integration.
- `project_id` (String) The ID of the Vercel project.

### Optional

- `team_id` (String) The ID of the Vercel team.Required when configuring a team resource if a default team has not been set in the provider.
