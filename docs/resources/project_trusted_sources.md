---
page_title: "vercel_project_trusted_sources Resource - terraform-provider-vercel"
subcategory: ""
description: |-
  Manages Deployment Protection Trusted Sources as a standalone setting on an existing Vercel project. It never creates or deletes the project, and writes only the project's trustedSources setting.
  This resource owns the project's complete list of trusted projects and external sources: applying it replaces any entries that are not in the configuration. Destroying it clears Trusted Sources, leaving the project intact. To stop managing Trusted Sources while keeping them, use a removed block with lifecycle { destroy = false }.
  If vercel_project manages the same project, set manage_trusted_sources = false there and remove its inline trusted_sources attribute. Only one resource may own Trusted Sources per project. Import adopts the existing configuration without changing it.
---

# vercel_project_trusted_sources (Resource)

Manages Deployment Protection Trusted Sources as a standalone setting on an existing Vercel project. It never creates or deletes the project, and writes only the project's trustedSources setting.

This resource owns the project's complete list of trusted projects and external sources: applying it replaces any entries that are not in the configuration. Destroying it clears Trusted Sources, leaving the project intact. To stop managing Trusted Sources while keeping them, use a removed block with lifecycle { destroy = false }.

If vercel_project manages the same project, set manage_trusted_sources = false there and remove its inline trusted_sources attribute. Only one resource may own Trusted Sources per project. Import adopts the existing configuration without changing it.

## Example Usage

```terraform
# Standalone: this manages Trusted Sources on a project that already exists.
# It does not create or own the project itself.
resource "vercel_project_trusted_sources" "existing" {
  team_id    = "team_example"
  project_id = "prj_existing"

  # Other Vercel projects in the same team that may call this project's
  # protected deployments.
  projects = [
    {
      project_id = "prj_frontend"
      label      = "Frontend"
    },
  ]

  # External OIDC sources, such as GitHub Actions.
  external_sources = [
    {
      issuer = "https://token.actions.githubusercontent.com"
      label  = "GitHub Actions"
      to     = { slugs = ["preview"] }
      claims = {
        aud        = ["https://vercel.com/example"]
        repository = ["example/app"]
      }
    },
  ]

  enable_vercel_ci_same_repository = true
}

# If vercel_project also manages this project, set
# manage_trusted_sources = false there and omit its inline trusted_sources.
```

## Standalone ownership and coexistence

This resource works with a project created outside Terraform: no `vercel_project` resource or project import is required. It writes only the project's `trustedSources` setting with `PATCH /v9/projects/{id}`, and never creates or deletes the project.

If Terraform also manages the project itself:

```terraform
resource "vercel_project" "app" {
  name                   = "existing-app"
  manage_trusted_sources = false
  # Omit trusted_sources here. Keep your other existing project configuration.
}

resource "vercel_project_trusted_sources" "app" {
  project_id = vercel_project.app.id
  projects = [
    { project_id = "prj_frontend" },
  ]
}
```

Set `manage_trusted_sources = false` before delegating ownership. It omits Trusted Sources from every project request, including follow-up requests during project creation, and stops `vercel_project` from reporting them as drift. It leaves the live configuration unchanged. Deployment protection (`manage_deployment_protection`) and Passport (`manage_passport`) have their own independent flags and standalone resources.

Do not manage Trusted Sources through multiple resources, multiple Terraform states, or inline attributes for the same project.

## Adoption and removal

This resource owns the project's complete lists of trusted projects and external sources. Each apply sends both lists, so entries that are not in the configuration are removed. To keep existing entries, import them first: import is read-only and adopts the full live configuration. Review the plan before applying.

Destroying the resource sends `trustedSources: null`, which clears trusted projects and external sources. The API preserves the stored `enable_vercel_ci_same_repository` value on clear; set it back to `true` before destroying if you disabled it and want the default restored. To stop managing Trusted Sources while keeping them, replace the resource block with:

```terraform
removed {
  from = vercel_project_trusted_sources.app
  lifecycle {
    destroy = false
  }
}
```

The `removed` block requires Terraform 1.7 or later. Changing the project ID or an explicitly configured team ID replaces the resource, clearing Trusted Sources on the old project and writing them to the new one.

## Same-repository Vercel CI access

`enable_vercel_ci_same_repository` controls whether same-team Vercel CI runs can access preview and production deployments built from the CI run's repository. The API treats a missing value as enabled. This resource writes the value only when it is configured; omitting it leaves the stored value unchanged and reports the live value.

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `project_id` (String) The ID of the existing project. No project is created by this resource.

### Optional

- `enable_vercel_ci_same_repository` (Boolean) Allow same-team Vercel CI runs to access preview and production deployments built from the same repository as the CI run. The API enables this when no value is stored. When omitted, the stored value is left unchanged, including on destroy.
- `external_sources` (Attributes Set) External sources that can reach this project's protected deployments using short-lived OIDC tokens. Omit to trust no external sources; removing every entry clears the list. (see [below for nested schema](#nestedatt--external_sources))
- `projects` (Attributes Set) Vercel projects in the same team that can reach this project's protected deployments. Omit to trust no other projects; removing every entry clears the list. (see [below for nested schema](#nestedatt--projects))
- `team_id` (String) The team ID. Defaults to the provider team.

### Read-Only

- `id` (String) The project ID.

<a id="nestedatt--external_sources"></a>
### Nested Schema for `external_sources`

Required:

- `claims` (Map of Set of String) Claims that must match on the OIDC token. Each key is a claim name, and each value is a set of accepted values. The API requires `aud` and issuer-specific identity claims.
- `issuer` (String) The OIDC issuer URL.
- `to` (Attributes) The target environments on this project that may be accessed. (see [below for nested schema](#nestedatt--external_sources--to))

Optional:

- `label` (String) A label or description for the trusted external source entry.

<a id="nestedatt--external_sources--to"></a>
### Nested Schema for `external_sources.to`

Optional:

- `preset` (String) Named environment preset. Currently only `all-custom` is supported.
- `slugs` (Set of String) System environment slugs (`production`, `preview`, `development`) or custom environment slugs.



<a id="nestedatt--projects"></a>
### Nested Schema for `projects`

Required:

- `project_id` (String) The trusted Vercel project ID.

Optional:

- `custom_allow` (Attributes Set) Optional overrides for default same-environment matching. Saved rules replace the API defaults for this trusted project. (see [below for nested schema](#nestedatt--projects--custom_allow))
- `label` (String) A label or description for the trusted project.

<a id="nestedatt--projects--custom_allow"></a>
### Nested Schema for `projects.custom_allow`

Required:

- `from` (Attributes) The source environments on the trusted project that are allowed to access the target environments. (see [below for nested schema](#nestedatt--projects--custom_allow--from))
- `to` (Attributes) The target environments on this project that may be accessed. (see [below for nested schema](#nestedatt--projects--custom_allow--to))

<a id="nestedatt--projects--custom_allow--from"></a>
### Nested Schema for `projects.custom_allow.from`

Optional:

- `preset` (String) Named environment preset. Currently only `all-custom` is supported.
- `slugs` (Set of String) System environment slugs (`production`, `preview`, `development`) or custom environment slugs.


<a id="nestedatt--projects--custom_allow--to"></a>
### Nested Schema for `projects.custom_allow.to`

Optional:

- `preset` (String) Named environment preset. Currently only `all-custom` is supported.
- `slugs` (Set of String) System environment slugs (`production`, `preview`, `development`) or custom environment slugs.

## Import

```shell
# Import the existing Trusted Sources configuration, without importing the project itself.
terraform import vercel_project_trusted_sources.existing team_example/prj_existing

# With a provider team configured, the project ID alone is sufficient.
terraform import vercel_project_trusted_sources.existing prj_existing
```
