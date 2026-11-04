# Uses the default team configured on the provider.
data "vercel_projects" "nextjs" {
  framework = "nextjs"
}

output "nextjs_project_ids" {
  value = keys(data.vercel_projects.nextjs.projects)
}

# An explicit team_id overrides the provider's default team.
# Protection filters are useful for audits. Avoid using a changing protection
# state as the for_each selection for resources that change that protection.
data "vercel_projects" "without_passport" {
  team_id          = "team_example"
  passport_enabled = false
}
