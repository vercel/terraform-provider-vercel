resource "vercel_access_group" "example" {
  name = "example-access-group"
}

resource "vercel_team_member" "example" {
  team_id = "team_xxxxxxxxxxxxxxxxxxxxxxxx"
  email   = "example@example.com"
  role    = "MEMBER"
}

resource "vercel_access_group_member" "example" {
  access_group_id = vercel_access_group.example.id
  user_id         = vercel_team_member.example.user_id
}
