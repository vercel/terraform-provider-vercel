# If importing into a personal account, or with a default team configured in
# the provider, simply use the trace_drain_id.
terraform import vercel_trace_drain.example drn_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Alternatively, you can import via team_id/trace_drain_id.
terraform import vercel_trace_drain.example team_xxxxxxxxxxxxxxxxxxxxxxxx/drn_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
