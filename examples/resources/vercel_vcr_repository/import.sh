# If importing into a personal account, or with a team configured on
# the provider, simply use the project_id and repository name.
# - project_id can be found in the project `settings` tab in the Vercel UI.
terraform import vercel_vcr_repository.example prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/my-repository

# Alternatively, you can import via the team_id, project_id and repository name.
# - team_id can be found in the team `settings` tab in the Vercel UI.
# - project_id can be found in the project `settings` tab in the Vercel UI.
terraform import vercel_vcr_repository.example team_xxxxxxxxxxxxxxxxxxxxxxxx/prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/my-repository
