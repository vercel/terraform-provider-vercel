resource "vercel_oauth_app" "example" {
  name = "My Example App"
  slug = "my-example-app"

  description   = "Lets users sign in to Example with their Vercel account."
  home_page_uri = "https://example.com"

  redirect_uris = [
    "https://example.com/api/auth/callback",
    "http://localhost:3000/api/auth/callback",
  ]

  # "openid" is always required; "offline_access" issues refresh tokens.
  scopes = ["openid", "email", "profile", "offline_access"]

  # Vercel REST API permissions the app's tokens may exercise (consented by
  # the user at sign-in).
  permissions = [
    "read:team",
    "read:project",
    "read-write:project",
    "read:deployment",
    "read-write:deployment",
  ]

  privacy_policy_url   = "https://example.com/privacy"
  terms_of_service_url = "https://example.com/terms"
}
