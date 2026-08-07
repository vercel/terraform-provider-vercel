resource "vercel_oauth_app" "example" {
  name = "My Example App"
  slug = "my-example-app"

  redirect_uris = ["https://example.com/api/auth/callback"]
  scopes        = ["openid", "email", "profile", "offline_access"]
}

# Vercel REST API permissions the app's tokens may exercise on the signed-in
# user's behalf, consented at sign-in. One grant resource per app.
resource "vercel_oauth_app_permissions" "example" {
  oauth_app_id = vercel_oauth_app.example.id

  permissions = [
    "read:team",
    "read:project",
    "read-write:project",
    "read:deployment",
    "read-write:deployment",
  ]
}
