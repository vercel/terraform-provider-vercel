# If importing into a personal account, or with a team configured on
# the provider, use the access_group_id and user_id.
terraform import vercel_access_group_member.example ag_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/xxxxxxxxxxxxxxxxxxxxxxxxxx

# If importing to a team, use the team_id, access_group_id and user_id.
terraform import vercel_access_group_member.example team_xxxxxxxxxxxxxxxxxxxxxxxx/ag_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/xxxxxxxxxxxxxxxxxxxxxxxxxx
