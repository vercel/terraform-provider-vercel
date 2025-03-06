resource "vercel_team_member" "by_user_id" {
  team_id = "team_xxxxxxxxxxxxxxxxxxxxxxxx"
  user_id = "uuuuuuuuuuuuuuuuuuuuuuuuuu"
  role    = "MEMBER"
}

resource "vercel_team_member" "by_email" {
  team_id = "team_xxxxxxxxxxxxxxxxxxxxxxxx"
  email   = "example@example.com"
  role    = "MEMBER"
}
