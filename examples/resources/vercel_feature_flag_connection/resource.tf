resource "vercel_project" "source" {
  name = "feature-flag-connection-source"
}

resource "vercel_project" "consumer" {
  name = "feature-flag-connection-consumer"
}

# Flags must be enabled on the source project before other projects can be
# connected to it. Creating a flag enables Flags on the project.
resource "vercel_feature_flag_definition" "example" {
  project_id = vercel_project.source.id
  key        = "checkout-redesign"
  kind       = "boolean"
  variant = [
    {
      id         = "off"
      value_bool = false
    },
    {
      id         = "on"
      value_bool = true
    },
  ]
}

# Lets the consumer project read the flags of the source project.
resource "vercel_feature_flag_connection" "example" {
  project_id          = vercel_project.source.id
  consumer_project_id = vercel_project.consumer.id

  depends_on = [vercel_feature_flag_definition.example]
}
