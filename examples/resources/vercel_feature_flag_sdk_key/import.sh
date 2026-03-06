# If importing into a personal account, or with a team configured on
# the provider, simply use the project ID and SDK key hash.
# - project_id can be found in the project `settings` tab in the Vercel UI.
# - hash_key can be found from the Flags SDK Keys UI or API.
terraform import vercel_feature_flag_sdk_key.example prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/sdk_xxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Alternatively, you can import via the team_id, project_id, and SDK key hash.
# - team_id can be found in the team `settings` tab in the Vercel UI.
# - project_id can be found in the project `settings` tab in the Vercel UI.
terraform import vercel_feature_flag_sdk_key.example team_xxxxxxxxxxxxxxxxxxxxxxxx/prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/sdk_xxxxxxxxxxxxxxxxxxxxxxxxxxxx
