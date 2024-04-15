resource "vercel_edge_config" "example" {
  name = "example"
}

resource "vercel_edge_config_schema" "example" {
  id         = vercel_edge_config.example.id
  definition = <<EOF
{
  "title": "Greeting",
  "type": "object",
  "properties": {
    "greeting": {
      "description": "A friendly greeting",
      "type": "string"
    }
  }
}
EOF
}
