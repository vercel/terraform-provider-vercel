# If importing into a personal account, or with a team configured on
# the provider, simply use the project ID and segment ID.
# - project_id can be found in the project `settings` tab in the Vercel UI.
# - segment_id can be found from the Flags API or by inspecting the segment in the Vercel UI.
terraform import vercel_feature_flag_segment.example prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/segment_xxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Alternatively, you can import via the team_id, project_id, and segment ID.
# - team_id can be found in the team `settings` tab in the Vercel UI.
# - project_id can be found in the project `settings` tab in the Vercel UI.
terraform import vercel_feature_flag_segment.example team_xxxxxxxxxxxxxxxxxxxxxxxx/prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/segment_xxxxxxxxxxxxxxxxxxxxxxxxxxxx
