resource "vercel_audit_log_drain" "example" {
  name = "security-audit-events"

  http {
    endpoint    = "https://siem.example.com/vercel"
    encoding    = "ndjson"
    compression = "gzip"

    headers = {
      Authorization = "Bearer my-secret-token"
    }
  }
}
