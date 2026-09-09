# If a default team is configured in the provider, use the alert_rule_id.
terraform import vercel_alert_rule.example ar_xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx

# Otherwise, import via team_id/alert_rule_id.
terraform import vercel_alert_rule.example team_xxxxxxxxxxxxxxxxxxxxxxxx/ar_xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
