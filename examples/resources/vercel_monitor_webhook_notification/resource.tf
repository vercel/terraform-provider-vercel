resource "vercel_webhook" "alerts" {
  endpoint = "https://example.com/vercel-alerts"
  events   = ["alerts.triggered"]
}

resource "vercel_monitor_webhook_notification" "alerts" {
  monitor_id = "ar_xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  webhook_id = vercel_webhook.alerts.id
}
