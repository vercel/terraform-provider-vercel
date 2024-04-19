// Use the vercel_endpoint_verification data source to work out the verification code needed to
// verify the log drain endpoint.
data "vercel_endpoint_verification" "example" {
}

resource "vercel_log_drain" "example" {
  delivery_format = "json"
  environments    = ["production"]
  headers = {
    some-key = "some-value"
  }
  project_ids   = [vercel_project.example.id]
  sampling_rate = 0.8
  secret        = "a_very_long_and_very_well_specified_secret"
  sources       = ["static"]
  endpoint      = "https://example.com/my-log-drain-endpoint"
}

resource "vercel_project" "example" {
  name = "example"
}
