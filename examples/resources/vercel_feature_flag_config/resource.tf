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

resource "vercel_feature_flag_config" "example" {
  project_id = vercel_project.example.id
  flag_id    = vercel_feature_flag_definition.example.id

  production = {
    enabled             = true
    default_variant_id  = "control"
    disabled_variant_id = "control"
  }

  preview = {
    enabled             = true
    default_variant_id  = "treatment"
    disabled_variant_id = "control"
  }

  development = {
    enabled             = false
    default_variant_id  = "treatment"
    disabled_variant_id = "control"
  }
}
