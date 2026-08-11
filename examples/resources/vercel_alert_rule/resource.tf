resource "vercel_project" "example" {
  name = "example-project"
}

# A built-in anomaly rule covering every project in the team.
resource "vercel_alert_rule" "team_usage" {
  name = "Team usage anomalies"
  alert_types = [{
    type = "usage_anomaly"
  }]
  sensitivity_level    = 3
  autosubscribe_owners = true
}

# A built-in anomaly rule limited to specific projects and to 5xx errors on a
# single route. Project scoping uses an OData expression.
resource "vercel_alert_rule" "checkout_errors" {
  name           = "Checkout 5xx anomalies"
  project_filter = "projectId in ('${vercel_project.example.id}')"
  alert_types = [{
    type   = "error_anomaly"
    filter = "statusGroup eq '5xx' and route eq '/api/checkout'"
  }]
  sensitivity_level            = 4
  autosubscribe_project_admins = true
}

# A custom alert rule that raises an alert when the hourly 5xx rate for a single
# project goes above 5%, ignoring hours with fewer than 20 requests.
resource "vercel_alert_rule" "checkout_error_rate" {
  name       = "Checkout error rate"
  project_id = vercel_project.example.id
  alert_types = [{
    type = "custom_alert"
  }]

  custom_alert = {
    event = "incomingRequest"
    rollups = {
      errors = {
        measure     = "count"
        aggregation = "sum"
        filter      = "httpStatus ge 500"
      }
      requests = {
        measure     = "count"
        aggregation = "sum"
      }
    }
    granularity       = "1h"
    trigger_type      = "threshold"
    trigger_operator  = "gt"
    trigger_threshold = 0.05
    min_threshold     = 20
    formula = {
      operator = "divide"
      left     = "errors"
      right    = "requests"
    }
  }
}

# A custom alert rule that raises an alert when per-route request volume is more
# than three standard deviations above its recent baseline.
resource "vercel_alert_rule" "request_volume_anomaly" {
  name       = "Request volume anomaly"
  project_id = vercel_project.example.id
  alert_types = [{
    type = "custom_alert"
  }]

  custom_alert = {
    event = "incomingRequest"
    rollups = {
      requests = {
        measure     = "count"
        aggregation = "sum"
      }
    }
    group_by          = ["route"]
    granularity       = "5m"
    trigger_type      = "anomaly"
    trigger_operator  = "gt"
    trigger_threshold = 3
  }
}
