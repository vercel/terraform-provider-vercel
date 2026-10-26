# If importing into a personal account, or with a team configured on
# the provider, simply use the log_drain_id.
# - log_drain_id can be found by querying the Vercel REST API (https://vercel.com/docs/rest-api/endpoints/logDrains#retrieves-a-list-of-all-the-log-drains).
terraform import vercel_log_drain.example ld_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Alternatively, you can import via the team_id and log_drain_id.
# - team_id can be found in the team `settings` tab in the Vercel UI.
# - log_drain_id can be found by querying the Vercel REST API (https://vercel.com/docs/rest-api/endpoints/logDrains#retrieves-a-list-of-all-the-log-drains).
terraform import vercel_log_drain.example team_xxxxxxxxxxxxxxxxxxxxxxxx/ld_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Prefix either import format with "write-only:" to omit the existing signing
# secret from Terraform state. Configure secret_wo after import. To replace the
# drain with that secret, also configure secret_wo_version or use -replace.
terraform import vercel_log_drain.example write-only:team_xxxxxxxxxxxxxxxxxxxxxxxx/ld_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
