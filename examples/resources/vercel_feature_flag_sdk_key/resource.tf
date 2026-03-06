resource "vercel_project" "example" {
  name = "feature-flag-sdk-key-example"
}

resource "vercel_feature_flag_sdk_key" "example" {
  project_id  = vercel_project.example.id
  environment = "production"
  type        = "server"
  label       = "backend-sdk"
}

resource "vercel_project_environment_variable" "example" {
  project_id = vercel_project.example.id
  target     = ["production"]
  key        = "FLAGS_CONNECTION_STRING"
  value      = vercel_feature_flag_sdk_key.example.connection_string
}
