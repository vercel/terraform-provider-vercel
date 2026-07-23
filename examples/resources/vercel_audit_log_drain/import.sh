# If importing with a default team configured in the provider, simply use the
# audit_log_drain_id.
terraform import vercel_audit_log_drain.example drn_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Alternatively, import using team_id/audit_log_drain_id.
terraform import vercel_audit_log_drain.example team_xxxxxxxxxxxxxxxxxxxxxxxx/drn_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
