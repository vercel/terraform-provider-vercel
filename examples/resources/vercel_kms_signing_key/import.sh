# If importing into a personal account, or with a team configured on
# the provider, use the issuer id and key id.
terraform import vercel_kms_signing_key.example iss_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/my-key-id

# Alternatively, you can import via the team_id, issuer id and key id.
# - team_id can be found in the team `settings` tab in the Vercel UI.
terraform import vercel_kms_signing_key.example team_xxxxxxxxxxxxxxxxxxxxxxxx/iss_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/my-key-id
