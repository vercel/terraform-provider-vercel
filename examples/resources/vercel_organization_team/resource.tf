resource "vercel_organization_team" "product" {
  organization_id = "org_xxxxxxxxxxxxxxxxxxxxxxxx"
  name            = "Product"
  slug            = "acme-product"
  billing_plan    = "enterprise"

  lifecycle {
    prevent_destroy = true
  }
}

resource "vercel_team_config" "product" {
  id                         = vercel_organization_team.product.id
  name                       = vercel_organization_team.product.name
  slug                       = vercel_organization_team.product.slug
  default_build_machine_type = "standard"
}

resource "vercel_project" "web" {
  team_id = vercel_organization_team.product.id
  name    = "web"
}
