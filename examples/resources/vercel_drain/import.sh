# If importing into a personal account, or with a team configured on
# the provider, simply use the drain ID.
terraform import vercel_drain.example drn_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Alternatively, you can import via the team_id and drain_id.
terraform import vercel_drain.example team_xxxxxxxxxxxxxxxxxxxxxxxx/drn_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
