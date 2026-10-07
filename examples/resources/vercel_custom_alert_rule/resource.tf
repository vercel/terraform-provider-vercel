data "vercel_project" "checkout" {
  name = "checkout"
}

resource "vercel_custom_alert_rule" "checkout_errors" {
  name       = "Checkout server errors"
  project_id = data.vercel_project.checkout.id
  severity   = "high"
  tags       = ["checkout", "payments"]

  evaluation = {
    window = "5m"
    query = {
      metrics = {
        errors = {
          metric      = "vercel.request.count"
          aggregation = "count"
          filter      = "httpStatus >= 500"
        }
      }
      outputs = ["errors"]
    }
  }

  trigger = {
    type      = "threshold"
    output    = "errors"
    operator  = "gt"
    threshold = 20
  }

  notification_settings = {
    enable_team_owner_notifications = true
  }
}

resource "vercel_alert_rule_slack_notification" "checkout_errors" {
  alert_rule_id    = vercel_custom_alert_rule.checkout_errors.id
  slack_channel_id = "C0123456789"
}

# Page on-call only when the agent investigation classifies the alert as Critical.
resource "vercel_alert_rule_slack_notification" "checkout_errors_oncall" {
  alert_rule_id          = vercel_custom_alert_rule.checkout_errors.id
  slack_channel_id       = "C0ONCALL00"
  minimum_severity_level = "critical"
}

resource "vercel_alert_rule_webhook_notification" "checkout_errors" {
  alert_rule_id = vercel_custom_alert_rule.checkout_errors.id
  webhook_id    = vercel_webhook.alerts.id
}

resource "vercel_webhook" "alerts" {
  endpoint = "https://example.com/vercel-alerts"
  events   = ["alerts.triggered"]
}
