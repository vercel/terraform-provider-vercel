# If a default team is configured in the provider, import a webhook notification link using:
terraform import vercel_alert_rule_notification.webhook ar_xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/webhook/hook_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Otherwise, prefix the import ID with the team ID:
terraform import vercel_alert_rule_notification.webhook team_xxxxxxxxxxxxxxxxxxxxxxxx/ar_xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/webhook/hook_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Import a Slack notification link using the alert rule, installation, and channel IDs:
terraform import vercel_alert_rule_notification.slack ar_xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/slack/icfg_xxxxxxxxxxxxxxxxxxxxxxxxxxxx/C0123456789
