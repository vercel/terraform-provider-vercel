# If importing into a personal account, or with a team configured on
# the provider, use the access_group_id and project_id.
terraform import vercel_access_group.example ag_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx

# If importing to a team, use the team_id, access_group_id and project_id.
terraform import vercel_access_group.example team_xxxxxxxxxxxxxxxxxxxxxxxx/ag_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx
