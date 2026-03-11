resource "vercel_project" "example" {
  name = "feature-flag-example"
}

resource "vercel_feature_flag_definition" "example" {
  project_id  = vercel_project.example.id
  key         = "checkout-redesign"
  description = "Controls the checkout experience"
  kind        = "string"
  variant = [
    {
      id           = "control"
      label        = "Control"
      value_string = "control"
    },
    {
      id           = "treatment"
      label        = "Treatment"
      value_string = "treatment"
    },
  ]
}
