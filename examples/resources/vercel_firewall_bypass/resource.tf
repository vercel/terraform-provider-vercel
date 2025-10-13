resource "vercel_project" "example" {
  name = "firewall-bypass-example"
}

resource "vercel_firewall_bypass" "bypass_targeted" {
  project_id = vercel_project.example.id

  source_ip = "5.6.7.8"
  # Any project domain assigned to the project can be used
  domain = "my-production-domain.com"
  note   = "Bypass rule for specific IP"
}

resource "vercel_firewall_bypass" "bypass_cidr" {
  project_id = vercel_project.example.id

  # CIDR ranges can be used as the source in bypass rules
  source_ip = "52.33.44.0/24"
  domain    = "my-production-domain.com"
  note      = "Bypass rule for CIDR range"
}

resource "vercel_firewall_bypass" "bypass_all" {
  project_id = vercel_project.example.id

  source_ip = "52.33.44.0/24"
  # the wildcard only domain can be used to apply a bypass
  # for all the _production_ domains assigned to the project.
  domain = "*"
}
