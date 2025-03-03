# If importing into a personal account, or with a team configured on the provider, simply use the record id.
# - the microfrontend ID can be taken from the microfrontend settings page
terraform import vercel_microfrontend_group.example mfe_xxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Alternatively, you can import via the team_id and microfrontend_id.
# - team_id can be found in the team `settings` tab in the Vercel UI.
# - the microfrontend ID can be taken from the microfrontend settings page
terraform import vercel_microfrontend_group.example team_xxxxxxxxxxxxxxxxxxxxxxxx/mfe_xxxxxxxxxxxxxxxxxxxxxxxxxxxx

