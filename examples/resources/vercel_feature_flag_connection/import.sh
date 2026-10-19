# If importing into a personal account, or with a team configured on
# the provider, simply use the source and consumer project IDs.
# - project_id can be found in the project `settings` tab in the Vercel UI.
terraform import vercel_feature_flag_connection.example prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/prj_yyyyyyyyyyyyyyyyyyyyyyyyyyyy

# Alternatively, you can import via the team_id, source project_id, and consumer project_id.
# - team_id can be found in the team `settings` tab in the Vercel UI.
terraform import vercel_feature_flag_connection.example team_xxxxxxxxxxxxxxxxxxxxxxxx/prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/prj_yyyyyyyyyyyyyyyyyyyyyyyyyyyy
