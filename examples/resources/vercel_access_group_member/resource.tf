resource "vercel_access_group" "example" {
  team_id = "team_xxxxxxxxxxxxxxxxxxxxxxxx"
  name    = "example-access-group"
}

resource "vercel_access_group_member" "example" {
  team_id         = "team_xxxxxxxxxxxxxxxxxxxxxxxx"
  access_group_id = vercel_access_group.example.id
  user_id         = "uuuuuuuuuuuuuuuuuuuuuuuuuu"
}
