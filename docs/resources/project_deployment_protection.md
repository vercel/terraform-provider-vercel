---
page_title: "vercel_project_deployment_protection Resource - terraform-provider-vercel"
subcategory: ""
description: |-
  Manages deployment protection as standalone settings on an existing Vercel project. It never creates or deletes the project, and does not manage builds, Git settings, Passport, bypasses, shareable links, or URL exceptions.
  Each configured object enables and owns that protection method. Objects never configured are left untouched. Removing a previously managed object disables that method. Destroying this resource disables only the methods recorded in its state, leaving the project intact. Use a removed block with lifecycle { destroy = false } to relinquish ownership without disabling protection.
  If vercel_project manages the same project, set manage_deployment_protection = false there and remove its inline vercel_authentication, password_protection, and trusted_ips attributes. Only one standalone resource may own these settings per project. Passwords are write-only, require Terraform 1.11 or later, and cannot be read or checked for drift. Import reads all enabled protection methods; omitted methods in the subsequent configuration will be disabled on apply.
---

# vercel_project_deployment_protection (Resource)

Manages deployment protection as standalone settings on an existing Vercel project. It never creates or deletes the project, and does not manage builds, Git settings, Passport, bypasses, shareable links, or URL exceptions.

Each configured object enables and owns that protection method. Objects never configured are left untouched. Removing a previously managed object disables that method. Destroying this resource disables only the methods recorded in its state, leaving the project intact. Use a removed block with lifecycle { destroy = false } to relinquish ownership without disabling protection.

If vercel_project manages the same project, set manage_deployment_protection = false there and remove its inline vercel_authentication, password_protection, and trusted_ips attributes. Only one standalone resource may own these settings per project. Passwords are write-only, require Terraform 1.11 or later, and cannot be read or checked for drift. Import reads all enabled protection methods; omitted methods in the subsequent configuration will be disabled on apply.

## Example Usage

```terraform
# Standalone: this manages protection on a project that already exists.
# It does not create or own the project itself.
resource "vercel_project_deployment_protection" "existing" {
  team_id    = "team_example"
  project_id = "prj_existing"

  vercel_authentication = {
    deployment_type = "all"
  }

  trusted_ips = {
    deployment_type = "all"
    addresses = [
      { value = "203.0.113.0/24", note = "Corporate VPN" }
    ]
  }

  # Password protection is not managed because it was never configured here.
}

# If vercel_project also manages this project, set
# manage_deployment_protection = false there and omit all three inline
# attributes: vercel_authentication, password_protection, and trusted_ips.
```

### Password protection without storing the password

```terraform
terraform {
  required_version = ">= 1.11.0"
}

variable "deployment_password" {
  type      = string
  sensitive = true
  ephemeral = true
}

# A separate existing project, protected by a password only.
resource "vercel_project_deployment_protection" "password" {
  project_id = "prj_password_project"

  password_protection = {
    deployment_type     = "preview"
    password_wo         = var.deployment_password
    password_wo_version = 1
  }
}
```

Supply `password_wo` and `password_wo_version` together when first enabling password protection. To rotate the password, change the input and increment the version. The version is stored in state; the password is not. Changing only the write-only input does not trigger an update. Terraform 1.11 or later is required for write-only arguments; an ephemeral variable also keeps the credential out of saved plans.

The API never returns the password, so password drift cannot be detected. If password protection is already enabled, you may omit both password arguments to adopt the existing password and manage only its deployment scope. Import never rotates it. Updates to other methods do not resend the password.

## Standalone ownership and coexistence

This resource works with a project created outside Terraform: no `vercel_project` resource or project import is required. Presence of each nested object enables that method; there is no `enabled` flag. Other protection methods remain untouched until they are configured or imported into this resource.

If Terraform also manages the project itself:

```terraform
resource "vercel_project" "app" {
  name                         = "existing-app"
  manage_deployment_protection = false
  # Omit vercel_authentication, password_protection, and trusted_ips here.
  # Keep your other existing project configuration.
}

resource "vercel_project_deployment_protection" "app" {
  project_id = vercel_project.app.id
  vercel_authentication = {
    deployment_type = "all"
  }
}
```

Set `manage_deployment_protection = false` before delegating ownership. It omits all three methods from project requests, including follow-up requests during project creation. It leaves their live configuration unchanged. `ignore_changes` alone does not stop broad project updates from writing observed settings. The project resource still owns the project's lifecycle and can delete it. Passport has its own independent `manage_passport` flag and standalone resource.

Do not manage the same settings through multiple standalone resources, multiple Terraform states, or inline attributes. Bypasses, shareable links, and public URL exceptions remain separate resources and can allow access even while these methods are enabled. This resource does not assert effective access policy or continuous team-wide enforcement.

## Adoption and removal

For **partial adoption**, add this standalone resource without an import block, configuring only the methods you want to manage. Applying it patches those methods on the existing project and leaves the rest alone. If you configure password protection without password arguments, an existing password is retained.

Import is the **adopt-all** path. Import reads all enabled Vercel Authentication, password, and Trusted IP methods. It never enables or disables them during import. Include each imported method you want to retain in the resource configuration: omitting an imported method plans to disable it. Review the complete plan before applying. This also applies when moving from inline protection: set the project's opt-out, remove its inline attributes, and import the existing settings into the standalone resource.

Removing a previously managed object disables that method. An object never configured or imported remains unmanaged. Destroying this resource disables only the methods in its state; it never deletes the project. To retain protection and stop managing it, replace the resource block with:

```terraform
removed {
  from = vercel_project_deployment_protection.app
  lifecycle {
    destroy = false
  }
}
```

The `removed` block requires Terraform 1.7 or later. Changing the project ID or an explicitly configured team ID replaces the attachment, disabling owned methods on the old project and enabling configured methods on the new project. Avoid `create_before_destroy` for replacements on the same project.

## Scope names and Trusted IP mode

Deployment scopes use API names consistently: `all`, `preview`, `prod_deployment_urls_and_all_previews`, and `all_except_custom_domains`. Trusted IPs also supports `production`. These correspond to the legacy project aliases `all_deployments`, `only_preview_deployments`, `standard_protection`, `standard_protection_new`, and `only_production_deployments` respectively.

Trusted IPs accepts IPv4 addresses and IPv4 CIDR ranges. The public project PATCH schema exposes addresses and deployment scope, but not `protectionMode`. This resource therefore reports `protection_mode` as read-only and does not change it; the server retains or chooses the mode. `additional` requires the allowlist alongside other configured protection; `exclusive` allows allowlisted visitors as an alternative. Review the reported mode when checking access policy.

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `project_id` (String) The ID of the existing project. No project is created by this resource.

### Optional

- `password_protection` (Attributes) Enable password protection. Omission leaves it unmanaged unless previously managed, in which case removal disables it. Import preserves the existing password. (see [below for nested schema](#nestedatt--password_protection))
- `team_id` (String) The team ID. Defaults to the provider team.
- `trusted_ips` (Attributes) Set the IPv4 allowlist and deployment scope. Removing this object disables Trusted IPs. The API's existing protection mode is reported but not changed. (see [below for nested schema](#nestedatt--trusted_ips))
- `vercel_authentication` (Attributes) Enable Vercel Authentication. Omission leaves it unmanaged unless previously managed, in which case removal disables it. (see [below for nested schema](#nestedatt--vercel_authentication))

### Read-Only

- `id` (String) The project ID.

<a id="nestedatt--password_protection"></a>
### Nested Schema for `password_protection`

Required:

- `deployment_type` (String) The deployment scope to protect, using API names: all, preview, prod_deployment_urls_and_all_previews, or all_except_custom_domains. Trusted IPs also supports production.

Optional:

- `password_wo` (String, Sensitive, [Write-only](https://developer.hashicorp.com/terraform/language/resources/ephemeral#write-only-arguments)) The password, never stored in state. Use an ephemeral variable to keep it out of saved plans. Required when enabling previously disabled password protection. Existing passwords are preserved when omitted. Change password_wo_version to rotate it.
- `password_wo_version` (Number) A non-secret rotation counter. Set with password_wo and increment to send a new password; changing only password_wo does not trigger an update.


<a id="nestedatt--trusted_ips"></a>
### Nested Schema for `trusted_ips`

Required:

- `addresses` (Attributes Set) Allowed IPv4 addresses or CIDR ranges. (see [below for nested schema](#nestedatt--trusted_ips--addresses))
- `deployment_type` (String) The deployment scope to protect, using API names: all, preview, prod_deployment_urls_and_all_previews, or all_except_custom_domains. Trusted IPs also supports production.

Read-Only:

- `protection_mode` (String) The API-reported Trusted IP mode: additional requires Trusted IPs alongside other protection; exclusive allows Trusted IPs as an alternative. The public PATCH schema does not expose mode updates, so this resource does not change it.

<a id="nestedatt--trusted_ips--addresses"></a>
### Nested Schema for `trusted_ips.addresses`

Required:

- `value` (String) An IPv4 address or CIDR range. IPv6 is not supported.

Optional:

- `note` (String) Optional description, up to 20 characters.



<a id="nestedatt--vercel_authentication"></a>
### Nested Schema for `vercel_authentication`

Required:

- `deployment_type` (String) The deployment scope to protect, using API names: all, preview, prod_deployment_urls_and_all_previews, or all_except_custom_domains. Trusted IPs also supports production.

## Import

```shell
# Import all currently enabled methods, without importing the project itself.
# Include every imported method you want to retain in your configuration.
terraform import vercel_project_deployment_protection.existing team_example/prj_existing

# With a provider team configured, the project ID alone is sufficient.
terraform import vercel_project_deployment_protection.existing prj_existing
```
