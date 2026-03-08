resource "vercel_project" "example" {
	name = "example-project"
}

resource "vercel_project_route" "example" {
	project_id = vercel_project.example.id
	name       = "redirect-legacy-docs"
	position = {
		placement = "start"
	}
	route = {
		src    = "/docs/:path*"
		dest   = "/guides/:path*"
		status = 308
	}
}

resource "vercel_project_route" "rewrite_eu_campaign" {
	project_id = vercel_project.example.id
	name       = "rewrite-eu-campaign"
	position = {
		placement          = "after"
		reference_route_id = vercel_project_route.example.id
	}
	route = {
		src  = "/promo"
		dest = "/campaigns/eu"
		has = [
			{
				type  = "header"
				key   = "x-region"
				value = "eu"
			}
		]
		missing = [
			{
				type  = "cookie"
				key   = "preview"
				value = "1"
			}
		]
	}
}
