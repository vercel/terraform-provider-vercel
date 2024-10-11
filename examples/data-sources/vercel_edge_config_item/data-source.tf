data "vercel_edge_config" "example" {
  id = "ecfg_xxxxxxxxxxxxxxxxxxxxxxxxxxxx"
}

data "vercel_edge_config_item" "test" {
  id  = data.vercel_edge_config.example.id
  key = "foobar"
}
