---
page_title: "Importing existing resources"
subcategory: ""
description: |-
  Import existing Vercel resources with the correct team ownership.
---

# Importing existing resources

Use the import format documented for the resource. Most resources accept an optional team ID prefix. For example:

```shell
terraform import vercel_project.example team_xxxxxxxxxxxxxxxxxxxxxxxx/prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx
terraform import vercel_project_domain.example team_xxxxxxxxxxxxxxxxxxxxxxxx/prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/example.com
```

You can also configure the default team on the provider and omit the team prefix where the resource's import format allows it:

```terraform
provider "vercel" {
  team = "team_xxxxxxxxxxxxxxxxxxxxxxxx"
}
```

The resource's `team_id` configuration is not an import scope. Supply the team in the import ID or provider configuration if the API requires a scoped request.

## Ownership discovery

An explicit import team takes precedence over the provider's default team for requests. The provider preserves ownership returned by the resource API. When neither import nor provider specifies a team and the child API does not return ownership, the provider reads the parent before importing:

| Resource | Ownership source |
| --- | --- |
| Project settings, domains, environment variables, custom environments, feature flags, firewall settings, deployment protection exceptions, routes, tracing, and VCR repositories/permissions | Project |
| Edge Config items, schemas, and tokens | Edge Config store |
| Access group members | Access group |
| Blob project connections | Blob store |
| KMS signing keys and project grants | KMS issuer |
| DNS records | Domain |
| Microfrontend group memberships | Project scope, then microfrontend group |

These lookups require permission to read the parent. If ownership cannot be determined, the import fails; supply the team explicitly or configure the provider team and retry. Ownership discovery does not bypass API access controls, and endpoints that require a team may reject an unscoped request.

Personal-account ownership remains null for resources that support it, including personal projects and DNS records. A user token without a team scope also legitimately has a null `team_id`.

## Resources with required scope

Some resources require an explicit team in their documented import format, such as team members and firewall bypass entries. Microfrontend groups require either an import team or a provider team because their API lists groups within a team. Team configuration uses the team itself as its import ID.

Empty identifiers, including an empty team prefix such as `/project_id`, are rejected. Use the documented form without the optional prefix instead.

## Existing imported state

Review the next plan after upgrading. Reads that already retrieve ownership can correct older state. Child resources whose ownership lookup runs only during import may need to be re-imported with the correct team to repair older null state. Do not apply an unexpected replacement solely to correct an imported `team_id`.
