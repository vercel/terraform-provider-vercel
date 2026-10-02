# If importing into a personal account, or with a team configured on
# the provider, simply use the domain name.
terraform import vercel_domain.example example.com

# Alternatively, you can import via the team_id and domain name.
# - team_id can be found in the team `settings` tab in the Vercel UI.
terraform import vercel_domain.example team_xxxxxxxxxxxxxxxxxxxxxxxx/example.com
