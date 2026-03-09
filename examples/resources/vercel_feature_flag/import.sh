# If importing into a personal account, or with a team configured on
# the provider, simply use the project ID and feature flag ID.
# - project_id can be found in the project `settings` tab in the Vercel UI.
# - flag_id can be found from the Flags API or by inspecting the flag in the Vercel UI.
terraform import vercel_feature_flag.example prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/flag_xxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Alternatively, you can import via the team_id, project_id, and feature flag ID.
# - team_id can be found in the team `settings` tab in the Vercel UI.
# - project_id can be found in the project `settings` tab in the Vercel UI.
terraform import vercel_feature_flag.example team_xxxxxxxxxxxxxxxxxxxxxxxx/prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/flag_xxxxxxxxxxxxxxxxxxxxxxxxxxxx
