---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "vercel_firewall_bypass Resource - terraform-provider-vercel"
subcategory: ""
description: |-
  Provides a Firewall Bypass Rule
  Firewall Bypass Rules configure sets of domains and ip address to prevent bypass Vercel's system mitigations for.  The hosts used in a bypass rule must be a production domain assigned to the associated project.  Requests that bypass system mitigations will incur usage.
---

# vercel_firewall_bypass (Resource)

Provides a Firewall Bypass Rule

Firewall Bypass Rules configure sets of domains and ip address to prevent bypass Vercel's system mitigations for.  The hosts used in a bypass rule must be a production domain assigned to the associated project.  Requests that bypass system mitigations will incur usage.

## Example Usage

```terraform
resource "vercel_project" "example" {
  name = "firewall-bypass-example"
}

resource "vercel_firewall_bypass" "bypass_targeted" {
  project_id = vercel_project.example.id

  source_ip = "5.6.7.8"
  # Any project domain assigned to the project can be used
  domain = "my-production-domain.com"
}

resource "vercel_firewall_bypass" "bypass_cidr" {
  project_id = vercel_project.example.id

  # CIDR ranges can be used as the source in bypass rules
  source_ip = "52.33.44.0/24"
  domain = "my-production-domain.com"
}

resource "vercel_firewall_bypass" "bypass_all" {
  project_id = vercel_project.example.id

  source_ip = "52.33.44.0/24"
  # the wildcard only domain can be used to apply a bypass
  # for all the _production_ domains assigned to the project.
  domain = "*"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `domain` (String) The domain to configure the bypass rule for.
- `project_id` (String) The ID of the Project to assign the bypass rule to
- `source_ip` (String) The source IP address to configure the bypass rule for.

### Optional

- `team_id` (String) The ID of the team the Project exists under. Required when configuring a team resource if a default team has not been set in the provider.

### Read-Only

- `id` (String) The identifier for the firewall bypass rule.

## Import

Import is supported using the following syntax:

```shell
terraform import vercel_firewall_bypass.example team_xxxxxxxxxxxxxxxxxxxxxxxx/prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx#mybypasshost.com#3.4.5.0/24


terraform import vercel_firewall_bypass.example team_xxxxxxxxxxxxxxxxxxxxxxxx/prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx#3.4.5.0/24
```
