data "vercel_connect_application" "passport" {
  team_id = "team_..."
  uid     = "oauth/company-sso"
}

resource "vercel_project" "internal_app" {
  team_id = data.vercel_connect_application.passport.team_id
  name    = "internal-app"

  passport = {
    connector_id = data.vercel_connect_application.passport.id
  }
}
