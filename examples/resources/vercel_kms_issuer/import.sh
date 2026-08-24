# If importing into a personal account, or with a team configured on
# the provider, simply use the issuer id.
# - issuer_id can be found via the vercel_kms_issuer data source or the API.
terraform import vercel_kms_issuer.example iss_xxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Alternatively, you can import via the team_id and issuer id.
# - team_id can be found in the team `settings` tab in the Vercel UI.
terraform import vercel_kms_issuer.example team_xxxxxxxxxxxxxxxxxxxxxxxx/iss_xxxxxxxxxxxxxxxxxxxxxxxxxxxx
