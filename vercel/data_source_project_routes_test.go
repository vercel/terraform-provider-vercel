package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_ProjectRoutesDataSource(t *testing.T) {
	nameSuffix := acctest.RandString(16)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy(testClient(t), "vercel_project.example", testTeam(t)),
		),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectRoutesDataSourceConfig(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectRouteExists(testClient(t), "vercel_project_route.anchor", testTeam(t)),
					testAccProjectRouteExists(testClient(t), "vercel_project_route.promo", testTeam(t)),
					testAccProjectRoutesOrder(testClient(t), "vercel_project.example", testTeam(t), "redirect-legacy", "rewrite-campaign"),
				),
			},
			{
				Config: cfg(testAccProjectRoutesDataSourceConfigWithDataSource(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectRouteCountInDataSource("data.vercel_project_routes.example", 2),
					resource.TestCheckResourceAttr("data.vercel_project_routes.example", "rules.0.name", "redirect-legacy"),
					resource.TestCheckResourceAttr("data.vercel_project_routes.example", "rules.0.route.src", "/legacy/:path*"),
					resource.TestCheckResourceAttr("data.vercel_project_routes.example", "rules.0.route.dest", "/modern/:path*"),
					resource.TestCheckResourceAttr("data.vercel_project_routes.example", "rules.0.route.status", "308"),
					resource.TestCheckResourceAttr("data.vercel_project_routes.example", "rules.1.name", "rewrite-campaign"),
					resource.TestCheckResourceAttr("data.vercel_project_routes.example", "rules.1.route.src", "/promo"),
					resource.TestCheckResourceAttr("data.vercel_project_routes.example", "rules.1.route.dest", "/campaign"),
					resource.TestCheckResourceAttr("data.vercel_project_routes.example", "rules.1.enabled", "true"),
					resource.TestCheckResourceAttr("data.vercel_project_routes.example", "rules.1.route.has.#", "1"),
					resource.TestCheckResourceAttr("data.vercel_project_routes.example", "rules.1.route.has.0.key", "x-region"),
					resource.TestCheckResourceAttr("data.vercel_project_routes.example", "rules.1.route.missing.#", "1"),
					resource.TestCheckResourceAttr("data.vercel_project_routes.example", "rules.1.route.missing.0.key", "preview"),
				),
			},
		},
	})
}

func testAccProjectRoutesDataSourceConfig(nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-project-routes-ds-%s"
}

resource "vercel_project_route" "anchor" {
	project_id = vercel_project.example.id
	name       = "redirect-legacy"
	position = {
		placement = "start"
	}
	route = {
		src    = "/legacy/:path*"
		dest   = "/modern/:path*"
		status = 308
	}
}

resource "vercel_project_route" "promo" {
	project_id = vercel_project.example.id
	name       = "rewrite-campaign"
	src_syntax = "equals"
	position = {
		placement          = "after"
		reference_route_id = vercel_project_route.anchor.id
	}
	route = {
		src  = "/promo"
		dest = "/campaign"
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
`, nameSuffix)
}

func testAccProjectRoutesDataSourceConfigWithDataSource(nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-project-routes-ds-%s"
}

resource "vercel_project_route" "anchor" {
	project_id = vercel_project.example.id
	name       = "redirect-legacy"
	position = {
		placement = "start"
	}
	route = {
		src    = "/legacy/:path*"
		dest   = "/modern/:path*"
		status = 308
	}
}

resource "vercel_project_route" "promo" {
	project_id = vercel_project.example.id
	name       = "rewrite-campaign"
	src_syntax = "equals"
	position = {
		placement          = "after"
		reference_route_id = vercel_project_route.anchor.id
	}
	route = {
		src  = "/promo"
		dest = "/campaign"
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

data "vercel_project_routes" "example" {
	project_id = vercel_project.example.id
}
`, nameSuffix)
}
