# If importing into a personal account, or with a team configured on
# the provider, simply use the access_group_id.
terraform import vercel_access_group.example ag_xxxxxxxxxxxxxxxxxxxxxxxxxxxx

# If importing to a team, use the team_id and access_group_id.
terraform import vercel_access_group.example team_xxxxxxxxxxxxxxxxxxxxxxxx/ag_xxxxxxxxxxxxxxxxxxxxxxxxxxxx
