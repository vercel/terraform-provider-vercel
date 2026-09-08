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

resource "vercel_alert_rule" "checkout_error_rate" {
  type = "custom"
  name = "Checkout error rate"
  rule_scope = {
    type       = "project"
    project_id = vercel_project.example.id
  }
  severity = "medium"
  evaluation = {
    window = "1h"
    query = {
      metrics = {
        errors = {
          metric      = "vercel.request.count"
          aggregation = "sum"
          filter      = "httpStatus>=500"
        }
        requests = {
          metric      = "vercel.request.count"
          aggregation = "sum"
        }
      }
      formulas = { formula = "errors / requests" }
      outputs  = ["formula"]
    }
  }
  trigger = {
    type      = "threshold"
    output    = "formula"
    operator  = "gt"
    threshold = 0.05
    minimum = {
      output    = "errors"
      threshold = 20
    }
  }
}
