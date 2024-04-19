# If importing into a personal account, or with a team configured on
# the provider, simply use the log_drain_id.
# - log_drain_id can be found by querying the Vercel REST API (https://vercel.com/docs/rest-api/endpoints/logDrains#retrieves-a-list-of-all-the-log-drains).
terraform import vercel_log_drain.example ecfg_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx

# Alternatively, you can import via the team_id and edge_config_id.
# - team_id can be found in the team `settings` tab in the Vercel UI.
# - log_drain_id can be found by querying the Vercel REST API (https://vercel.com/docs/rest-api/endpoints/logDrains#retrieves-a-list-of-all-the-log-drains).
terraform import vercel_log_drain.example team_xxxxxxxxxxxxxxxxxxxxxxxx/ecfg_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
