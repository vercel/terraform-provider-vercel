# If importing into a personal account, or with a team configured on
# the provider, use the project_id, repository name and granted team ID.
# - project_id can be found in the project `settings` tab in the Vercel UI.
# - granted_team_id is the ID of the team the repository is shared with.
terraform import vercel_vcr_repository_permission.example prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/my-repository/team_xxxxxxxxxxxxxxxxxxxxxxxx

# Alternatively, you can import via the owning team_id, project_id,
# repository name and granted team ID.
# - team_id can be found in the team `settings` tab in the Vercel UI.
# - project_id can be found in the project `settings` tab in the Vercel UI.
# - granted_team_id is the ID of the team the repository is shared with.
terraform import vercel_vcr_repository_permission.example team_xxxxxxxxxxxxxxxxxxxxxxxx/prj_xxxxxxxxxxxxxxxxxxxxxxxx/my-repository/team_yyyyyyyyyyyyyyyyyyyyyyyy
