# If a default team is configured in the provider, import using:
terraform import vercel_alert_rule_webhook_notification.alerts ar_xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/hook_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Otherwise, prefix the import ID with the team ID:
terraform import vercel_alert_rule_webhook_notification.alerts team_xxxxxxxxxxxxxxxxxxxxxxxx/ar_xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/hook_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
