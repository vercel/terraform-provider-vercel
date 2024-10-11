resource "vercel_edge_config" "example" {
  name = "example"
}

resource "vercel_edge_config_item" "example" {
  edge_config_id = vercel_edge_config.example.id
  key            = "foobar"
  value          = "baz"
}
