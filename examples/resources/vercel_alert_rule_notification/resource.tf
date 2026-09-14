resource "vercel_alert_rule_notification" "slack" {
  alert_rule_id         = "ar_xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  slack_installation_id = "icfg_xxxxxxxxxxxxxxxxxxxxxxxxxxxx"
  slack_channel_id      = "C0123456789"
}

resource "vercel_webhook" "alerts" {
  endpoint = "https://example.com/vercel-alerts"
  events   = ["alerts.triggered"]
}

resource "vercel_alert_rule_notification" "webhook" {
  alert_rule_id = "ar_xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  webhook_id    = vercel_webhook.alerts.id
}
