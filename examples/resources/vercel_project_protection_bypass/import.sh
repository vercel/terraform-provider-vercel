# If importing into a personal account, or with a team configured on
# the provider, use the project ID and bypass secret.
# - project_id can be found in the project `settings` tab in the Vercel UI.
# - secret is the 32-character bypass value (the map key under `protectionBypass` in the API).
terraform import vercel_project_protection_bypass.example prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/abcdefghijklmnopqrstuvwxyz123456

# Alternatively, you can import via team_id, project_id, and the bypass secret.
# - team_id can be found in the team `settings` tab in the Vercel UI.
terraform import vercel_project_protection_bypass.example team_xxxxxxxxxxxxxxxxxxxxxxxx/prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/abcdefghijklmnopqrstuvwxyz123456
