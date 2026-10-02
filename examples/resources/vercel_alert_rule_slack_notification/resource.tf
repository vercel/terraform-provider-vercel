resource "vercel_alert_rule_slack_notification" "alerts" {
  alert_rule_id    = "ar_xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  slack_channel_id = "C0123456789"
}
