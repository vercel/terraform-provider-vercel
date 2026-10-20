resource "vercel_organisation_team" "product" {
  organisation_id = "org_xxxxxxxxxxxxxxxxxxxxxxxx"
  name            = "Product"
  slug            = "acme-product"
  billing_plan    = "enterprise"

  lifecycle {
    prevent_destroy = true
  }
}

resource "vercel_team_config" "product" {
  id                         = vercel_organisation_team.product.id
  default_build_machine_type = "standard"
}

resource "vercel_project" "web" {
  team_id = vercel_organisation_team.product.id
  name    = "web"
}
