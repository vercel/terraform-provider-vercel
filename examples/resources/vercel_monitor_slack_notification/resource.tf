resource "vercel_monitor_slack_notification" "alerts" {
  monitor_id       = "ar_xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  slack_channel_id = "C0123456789"
}
