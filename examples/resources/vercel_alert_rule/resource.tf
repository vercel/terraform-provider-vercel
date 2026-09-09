resource "vercel_project" "example" {
  name = "example-project"
}

resource "vercel_alert_rule" "checkout_errors" {
  type = "built-in"
  name = "Checkout 5xx anomalies"
  rule_scope = {
    type        = "include"
    project_ids = [vercel_project.example.id]
  }
  triggers = [{
    type   = "error_anomaly"
    filter = "statusGroup:5xx AND route:/api/checkout"
  }]
  match_minimum_severity_level = "high"
  notification_settings = {
    enable_team_owner_notifications = true
  }
}
