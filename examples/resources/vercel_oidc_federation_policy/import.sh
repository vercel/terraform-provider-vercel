# If importing with a team configured on the provider, use the policy ID.
terraform import vercel_oidc_federation_policy.example pol_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Otherwise, use the team ID and policy ID.
terraform import vercel_oidc_federation_policy.example team_xxxxxxxxxxxxxxxxxxxxxxxx/pol_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
