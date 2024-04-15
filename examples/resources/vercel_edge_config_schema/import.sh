# If importing into a personal account, or with a team configured on
# the provider, simply use the edge config id.
# - edge_config_id can be found by navigating to the Edge Config in the Vercel UI. It should begin with `ecfg_`.
terraform import vercel_edge_config.example ecfg_xxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Alternatively, you can import via the team_id and edge_config_id.
# - team_id can be found in the team `settings` tab in the Vercel UI.
# - edge_config_id can be found by navigating to the Edge Config in the Vercel UI. It should begin with `ecfg_`.
terraform import vercel_edge_config.example team_xxxxxxxxxxxxxxxxxxxxxxxx/ecfg_xxxxxxxxxxxxxxxxxxxxxxxxxxxx
