# Import via the team_id and project_id.
# team_id can be found in the team `settings` tab in the Vercel UI.
# project_id can be found in the project `settings` tab in the Vercel UI.
terraform import vercel_project.example team_xxxxxxxxxxxxxxxxxxxxxxxx/prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx

# If importing without a team, simply use the project ID.
terraform import vercel_project.personal_example prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx
