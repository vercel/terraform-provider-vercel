resource "vercel_edge_config" "example" {
  name = "example"
}

resource "vercel_edge_config_item" "example" {
  edge_config_id = vercel_edge_config.example.id
  key            = "flags"
  value_json     = {
    featureA = true
    nested   = { a = 1, b = [1, 2, 3] }
  }
}
