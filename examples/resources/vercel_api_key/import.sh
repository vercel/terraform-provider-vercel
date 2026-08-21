# API keys can be imported by team and API key ID.
terraform import vercel_api_key.ai_gateway team_xxxxxxxxxxxxxxxxxxxx/key_xxxxxxxxxxxxxxxxxxxx

# Or when a default team is configured on the provider, by API key ID only.
terraform import vercel_api_key.ai_gateway key_xxxxxxxxxxxxxxxxxxxx
