// Use the vercel_endpoint_verification data source to work out the verification code needed to
// verify the log drain endpoint.
data "vercel_endpoint_verification" "example" {
}

ephemeral "random_password" "log_drain_secret" {
  length  = 64
  special = false
}

resource "vercel_log_drain" "example" {
  delivery_format = "json"
  environments    = ["production"]
  headers = {
    some-key = "some-value"
  }
  project_ids       = [vercel_project.example.id]
  sampling_rate     = 0.8
  secret_wo         = ephemeral.random_password.log_drain_secret.result
  # Optional: increment to replace the Log Drain when rotating the secret.
  secret_wo_version = 1
  sources           = ["static"]
  endpoint          = "https://example.com/my-log-drain-endpoint"
}

resource "vercel_project" "example" {
  name = "example"
}
