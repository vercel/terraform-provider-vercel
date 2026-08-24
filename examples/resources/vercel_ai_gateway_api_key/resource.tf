resource "vercel_ai_gateway_api_key" "example" {
  name = "workflow-github-actions"
}

# An API key with an expiration and a monthly spend quota.
resource "vercel_ai_gateway_api_key" "with_quota" {
  name       = "production"
  expires_at = 1767225600000 # January 1, 2026

  ai_gateway_quota = {
    limit_amount     = 500
    refresh_period   = "monthly"
    alert_thresholds = [50, 75, 100]
  }
}

output "ai_gateway_api_key" {
  value     = vercel_ai_gateway_api_key.example.api_key_string
  sensitive = true
}
