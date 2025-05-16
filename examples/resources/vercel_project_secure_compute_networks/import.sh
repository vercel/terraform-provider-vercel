# If importing with a team configured on the provider, simply use the project ID.
# - project_id can be found in the project `settings` tab in the Vercel UI.
terraform import vercel_project_secure_compute_networks.example prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Alternatively, you can import via the team_id and project_id.
# - team_id can be found in the team `settings` tab in the Vercel UI.
# - project_id can be found in the project `settings` tab in the Vercel UI.
terraform import vercel_project_secure_compute_networks.example team_xxxxxxxxxxxxxxxxxxxxxxxx/prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx
