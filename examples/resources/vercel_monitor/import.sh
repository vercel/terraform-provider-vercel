# If a default team is configured in the provider, use the monitor_id.
terraform import vercel_monitor.example ar_xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx

# Otherwise, import via team_id/monitor_id.
terraform import vercel_monitor.example team_xxxxxxxxxxxxxxxxxxxxxxxx/ar_xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
