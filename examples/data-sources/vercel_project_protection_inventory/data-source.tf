data "vercel_project_protection_inventory" "example" {
  project_id = "prj_example"
  team_id    = "team_example"
}

output "protection_access" {
  value = data.vercel_project_protection_inventory.example.entries
}
