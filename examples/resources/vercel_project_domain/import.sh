# If importing into a personal account, or with a team configured on
# the provider, simply use the project ID and domain.
# - project_id can be found in the project `settings` tab in the Vercel UI.
terraform import vercel_project_domain.example prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/example.com

# Alternatively, you can import via the team_id, project_id and domain name.
# - team_id can be found in the team `settings` tab in the Vercel UI.
# - project_id can be found in the project `settings` tab in the Vercel UI.
terraform import vercel_project_domain.example team_xxxxxxxxxxxxxxxxxxxxxxxx/prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/example.com
