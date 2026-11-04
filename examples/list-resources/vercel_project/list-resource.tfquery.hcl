# Uses the default team configured on the provider.
# Run: terraform query -generate-config-out=generated.tf
# Review the generated resource and import blocks before terraform plan/apply.
list "vercel_project" "nextjs" {
  provider = vercel
  limit    = 10000

  config {
    framework = "nextjs"
    # team_id = "team_example" # Optional override of the provider's team.
  }
}
