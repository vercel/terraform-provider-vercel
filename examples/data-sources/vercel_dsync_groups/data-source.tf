data "vercel_dsync_groups" "example" {
  team_id = "team_xxxxxxxxxxxxxxxxxxxxxxxxxxxx"
}

resource "vercel_access_group" "contractor" {
  team_id     = "team_xxxxxxxxxxxxxxxxxxxxxxxxxxxx"
  name        = "contractor"
  description = "Access group for contractors"
}

resource "vercel_team_config" "example" {
  id   = "team_xxxxxxxxxxxxxxxxxxxxxxxxxxxx"
  saml = {
    enforced = true
    roles = {
      lookup(vercel_dsync_groups.example.map, "admin") = {
        role = "OWNER"
      }
      lookup(vercel_dsync_groups.example.map, "finance") = {
        role = "BILLING"
      }
      lookup(vercel_dsync_groups.example.map, "contractor") = {
        role            = "CONTRIBUTOR"
        access_group_id = vercel_access_group.contractor.id
      }
    }
  }
}
