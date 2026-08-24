# AI Gateway API keys can be imported by team and API key ID.
terraform import vercel_ai_gateway_api_key.example team_xxxxxxxxxxxxxxxxxxxx/key_xxxxxxxxxxxxxxxxxxxx

# Or when a default team is configured on the provider, by API key ID only.
terraform import vercel_ai_gateway_api_key.example key_xxxxxxxxxxxxxxxxxxxx
