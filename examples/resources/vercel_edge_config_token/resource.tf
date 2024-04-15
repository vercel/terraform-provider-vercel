resource "vercel_edge_config" "example" {
  name = "example"
}

resource "vercel_project" "example" {
  name = "edge-config-example"
}

resource "vercel_edge_config_token" "example" {
  edge_config_id = vercel_edge_config.example.id
  label          = "example token"
}

resource "vercel_project_environment_variable" "example" {
  project_id = vercel_project.example.id
  target     = ["production", "preview", "development"]
  key        = "EDGE_CONFIG"
  value      = vercel_edge_config_token.example.connection_string
}
