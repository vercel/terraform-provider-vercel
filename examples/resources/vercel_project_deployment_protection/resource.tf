# Standalone: this manages protection on a project that already exists.
# It does not create or own the project itself.
resource "vercel_project_deployment_protection" "existing" {
  team_id    = "team_example"
  project_id = "prj_existing"

  vercel_authentication = {
    deployment_type = "all"
  }

  trusted_ips = {
    deployment_type = "all"
    addresses = [
      { value = "203.0.113.0/24", note = "Corporate VPN" }
    ]
  }

  # Password protection is not managed because it was never configured here.
}

# If vercel_project also manages this project, set
# manage_deployment_protection = false there and omit all three inline
# attributes: vercel_authentication, password_protection, and trusted_ips.
