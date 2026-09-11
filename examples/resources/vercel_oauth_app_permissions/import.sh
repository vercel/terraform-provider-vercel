# If importing into a personal account, or with a team configured on
# the provider, simply use the OAuth App's client_id.
terraform import vercel_oauth_app_permissions.example cl_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# If importing to a team, use the team_id and client_id.
terraform import vercel_oauth_app_permissions.example team_xxxxxxxxxxxxxxxxxxxxxxxx/cl_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
