# If importing into a personal account, or with a team configured on
# the provider, use the issuer id and project id.
terraform import vercel_kms_issuer_policy.example iss_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Alternatively, you can import via the team_id, issuer id and project id.
# - team_id can be found in the team `settings` tab in the Vercel UI.
terraform import vercel_kms_issuer_policy.example team_xxxxxxxxxxxxxxxxxxxxxxxx/iss_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/prj_xxxxxxxxxxxxxxxxxxxxxxxxxxxx
