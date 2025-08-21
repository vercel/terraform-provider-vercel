# Recommended: Use email to add team members
resource "vercel_team_member" "example" {
  team_id = "team_xxxxxxxxxxxxxxxxxxxxxxxx"
  email   = "example@example.com"
  role    = "MEMBER"
}
