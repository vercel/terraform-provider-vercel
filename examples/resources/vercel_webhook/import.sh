# If importing into a personal account, or with a team configured on the
# provider, use the webhook ID.
terraform import vercel_webhook.example hook_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Alternatively, import using the team ID and webhook ID.
terraform import vercel_webhook.example team_xxxxxxxxxxxxxxxxxxxxxxxx/hook_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Vercel does not return an existing webhook's signing secret, so the imported
# secret value will be null.
