resource "vercel_project" "example" {
  name = "example-project"
}

data "vercel_secure_compute_network" "example" {
  name = "Example Network"
}

resource "vercel_project_secure_compute_networks" "example" {
  project_id = vercel_project.example.id
  secure_compute_networks = [
    {
      environment    = "production"
      network_id     = data.vercel_secure_compute_network.example.id
      passive        = false
      builds_enabled = true
    }
  ]
}
