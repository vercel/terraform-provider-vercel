resource "vercel_ai_gateway_api_key" "example" {
  name = "workflow-github-actions"
}

output "ai_gateway_api_key" {
  value     = vercel_ai_gateway_api_key.example.api_key_string
  sensitive = true
}
