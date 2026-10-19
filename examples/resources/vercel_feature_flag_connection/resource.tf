resource "vercel_project" "source" {
  name = "feature-flag-connection-source"
}

resource "vercel_project" "consumer" {
  name = "feature-flag-connection-consumer"
}

# Lets the consumer project read the flags of the source project.
resource "vercel_feature_flag_connection" "example" {
  project_id          = vercel_project.source.id
  consumer_project_id = vercel_project.consumer.id
}
