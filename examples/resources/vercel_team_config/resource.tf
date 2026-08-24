data "vercel_file" "example" {
  path = "example/avatar.png"
}

resource "vercel_team_config" "example" {
  id                                    = "team_xxxxxxxxxxxxxxxxxxxxxxxx"
  avatar                                = data.vercel_file.example.file
  name                                  = "Vercel terraform example"
  slug                                  = "vercel-terraform-example"
  description                           = "Vercel Terraform Example"
  # Paid teams must be eligible for Basic build machine routing.
  default_build_machine_type            = "basic"
  sensitive_environment_variable_policy = "off"
  remote_caching = {
    enabled = true
  }
  enable_preview_feedback         = "off"
  enable_production_feedback      = "off"
  hide_ip_addresses               = true
  hide_ip_addresses_in_log_drains = true
}
