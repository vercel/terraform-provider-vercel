resource "vercel_api_key" "ai_gateway" {
  name    = "workflow-github-actions"
  purpose = "ai-gateway"
}

output "ai_gateway_api_key" {
  value     = vercel_api_key.ai_gateway.api_key_string
  sensitive = true
}
