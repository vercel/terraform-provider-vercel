# If importing into a personal account, or with a team configured on
# the provider, simply use the edge config id and the key of the item to import.
# - edge_config_id can be found by navigating to the Edge Config in the Vercel UI. It should begin with `ecfg_`.
# - key is the key of teh item to import.
terraform import vercel_edge_config.example ecfg_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/example_key

# Alternatively, you can import via the team_id, edge_config_id and the key of the item to import.
# - team_id can be found in the team `settings` tab in the Vercel UI.
# - edge_config_id can be found by navigating to the Edge Config in the Vercel UI. It should begin with `ecfg_`.
# - key is the key of the item to import.
terraform import vercel_edge_config.example team_xxxxxxxxxxxxxxxxxxxxxxxx/ecfg_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/example_key
