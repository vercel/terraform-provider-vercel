data "vercel_edge_config" "example" {
  id = "ecfg_xxxxxxxxxxxxxxxxxxxxxxxxxxxx"
}

# Read a string item
data "vercel_edge_config_item" "string_item" {
  id  = data.vercel_edge_config.example.id
  key = "foobar"
}

# Read a JSON item
data "vercel_edge_config_item" "json_item" {
  id  = data.vercel_edge_config.example.id
  key = "flags"
}
