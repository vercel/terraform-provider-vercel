resource "vercel_project" "example" {
  name           = "example_project"
  framework      = "create-react-app"
  root_directory = "packages/ui"

  environment = [
    {
      key    = "bar"
      value  = "baz"
      target = ["preview"]
    }
  ]
}
