resource "vercel_user_token" "example" {
  name = "example-token"
}

resource "vercel_user_token" "example_expiring" {
  name       = "example-expiring-token"
  expires_at = 1767225600000
}
