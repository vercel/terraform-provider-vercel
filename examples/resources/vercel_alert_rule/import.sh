# If importing into a personal account, or with a default team configured in
# the provider, simply use the alert_rule_id.
terraform import vercel_alert_rule.example ar_xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx

# Alternatively, you can import via team_id/alert_rule_id.
terraform import vercel_alert_rule.example team_xxxxxxxxxxxxxxxxxxxxxxxx/ar_xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
