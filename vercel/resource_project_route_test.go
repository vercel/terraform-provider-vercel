package vercel_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

func getProjectRouteImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}

		return fmt.Sprintf("%s/%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.Attributes["project_id"], rs.Primary.ID), nil
	}
}

func testAccGetLiveProjectRoutes(ctx context.Context, testClient *client.Client, projectID, teamID string) (client.ProjectRoutingRulesResponse, error) {
	versions, err := testClient.GetProjectRouteVersions(ctx, projectID, teamID)
	if err != nil {
		return client.ProjectRoutingRulesResponse{}, err
	}

	liveVersionID := ""
	for _, version := range versions {
		if version.IsLive {
			liveVersionID = version.ID
			break
		}
	}

	return testClient.GetProjectRoutingRules(ctx, projectID, teamID, liveVersionID)
}

func testAccProjectRouteExists(testClient *client.Client, n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		response, err := testAccGetLiveProjectRoutes(context.TODO(), testClient, rs.Primary.Attributes["project_id"], teamID)
		if err != nil {
			return err
		}

		for _, route := range response.Routes {
			if route.ID == rs.Primary.ID {
				return nil
			}
		}

		return fmt.Errorf("route %s not found in live project routes", rs.Primary.ID)
	}
}

func testAccProjectRoutesOrder(testClient *client.Client, n, teamID string, expectedNames ...string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		response, err := testAccGetLiveProjectRoutes(context.TODO(), testClient, rs.Primary.ID, teamID)
		if err != nil {
			return err
		}

		if len(response.Routes) != len(expectedNames) {
			return fmt.Errorf("expected %d live project routes, got %d", len(expectedNames), len(response.Routes))
		}

		for i, expectedName := range expectedNames {
			if response.Routes[i].Name != expectedName {
				return fmt.Errorf("expected route %d to be %q, got %q", i, expectedName, response.Routes[i].Name)
			}
		}

		return nil
	}
}

func testAccCaptureProjectRouteID(n string, destination *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		*destination = rs.Primary.ID
		return nil
	}
}

func testAccProjectRouteIDMatches(n string, expected *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID != *expected {
			return fmt.Errorf("expected route ID %q, got %q", *expected, rs.Primary.ID)
		}

		return nil
	}
}

func testAccProjectRouteIDChanged(n string, previous *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == *previous {
			return fmt.Errorf("expected route ID to change from %q", *previous)
		}

		return nil
	}
}

func testAccProjectRouteCountInDataSource(n string, expected int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		count, err := strconv.Atoi(rs.Primary.Attributes["rules.#"])
		if err != nil {
			return fmt.Errorf("unable to parse rules count: %w", err)
		}

		if count != expected {
			return fmt.Errorf("expected %d rules, got %d", expected, count)
		}

		return nil
	}
}

func TestAcc_ProjectRoute(t *testing.T) {
	resourceName := "vercel_project_route.promo"
	projectResourceName := "vercel_project.example"
	nameSuffix := acctest.RandString(16)
	originalRouteID := ""

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy(testClient(t), projectResourceName, testTeam(t)),
		),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectRouteConfig(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(projectResourceName, "id"),
					testAccProjectRouteExists(testClient(t), "vercel_project_route.anchor", testTeam(t)),
					testAccProjectRouteExists(testClient(t), resourceName, testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project_route.anchor", "position.placement", "start"),
					resource.TestCheckResourceAttr(resourceName, "position.placement", "after"),
					resource.TestCheckResourceAttr(resourceName, "name", "rewrite-campaign"),
					resource.TestCheckResourceAttr(resourceName, "route.dest", "/campaign"),
					resource.TestCheckResourceAttr(resourceName, "route.has.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "route.has.0.value", "eu"),
					testAccProjectRoutesOrder(testClient(t), projectResourceName, testTeam(t), "redirect-legacy", "rewrite-campaign"),
					testAccCaptureProjectRouteID(resourceName, &originalRouteID),
				),
			},
			{
				Config: cfg(testAccProjectRouteConfigUpdated(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectRouteExists(testClient(t), resourceName, testTeam(t)),
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "route.dest", "/campaign/eu"),
					resource.TestCheckResourceAttr(resourceName, "route.case_sensitive", "false"),
					resource.TestCheckResourceAttr(resourceName, "route.has.0.value", "uk"),
					resource.TestCheckResourceAttr(resourceName, "position.placement", "after"),
					testAccProjectRouteIDMatches(resourceName, &originalRouteID),
					testAccProjectRoutesOrder(testClient(t), projectResourceName, testTeam(t), "redirect-legacy", "rewrite-campaign"),
				),
			},
			{
				Config: cfg(testAccProjectRouteConfigMoved(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectRouteExists(testClient(t), resourceName, testTeam(t)),
					resource.TestCheckResourceAttr(resourceName, "position.placement", "start"),
					testAccProjectRouteIDChanged(resourceName, &originalRouteID),
					testAccProjectRoutesOrder(testClient(t), projectResourceName, testTeam(t), "rewrite-campaign", "redirect-legacy"),
				),
			},
			{
				Config: cfg(testAccProjectRouteConfigWithoutPromo(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectRouteExists(testClient(t), "vercel_project_route.anchor", testTeam(t)),
					testAccProjectRoutesOrder(testClient(t), projectResourceName, testTeam(t), "redirect-legacy"),
				),
			},
		},
	})
}

func TestAcc_ProjectRouteImport(t *testing.T) {
	nameSuffix := acctest.RandString(16)
	resourceName := "vercel_project_route.example"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy(testClient(t), "vercel_project.example", testTeam(t)),
		),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectRouteImportConfig(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectRouteExists(testClient(t), resourceName, testTeam(t)),
					resource.TestCheckResourceAttr(resourceName, "name", "redirect-legacy"),
					resource.TestCheckResourceAttr(resourceName, "position.placement", "start"),
				),
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateIdFunc:                    getProjectRouteImportID(resourceName),
				ImportStateVerifyIdentifierAttribute: "project_id",
				ImportStateVerifyIgnore:              []string{"position"},
			},
		},
	})
}

func testAccProjectRouteConfig(nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-project-route-%s"
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
	position = {
		placement          = "after"
		reference_route_id = vercel_project_route.anchor.id
	}
	route = {
		src            = "/promo"
		dest           = "/campaign"
		case_sensitive = true
		has = [
			{
				type  = "header"
				key   = "x-region"
				value = "eu"
			}
		]
	}
}
`, nameSuffix)
}

func testAccProjectRouteConfigUpdated(nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-project-route-%s"
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
	project_id  = vercel_project.example.id
	name        = "rewrite-campaign"
	enabled     = false
	description = "Route promo traffic to the EU landing page"
	position = {
		placement          = "after"
		reference_route_id = vercel_project_route.anchor.id
	}
	route = {
		src            = "/promo"
		dest           = "/campaign/eu"
		case_sensitive = false
		has = [
			{
				type  = "header"
				key   = "x-region"
				value = "uk"
			}
		]
	}
}
`, nameSuffix)
}

func testAccProjectRouteConfigMoved(nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-project-route-%s"
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
	project_id  = vercel_project.example.id
	name        = "rewrite-campaign"
	enabled     = false
	description = "Route promo traffic to the EU landing page"
	position = {
		placement = "start"
	}
	route = {
		src            = "/promo"
		dest           = "/campaign/eu"
		case_sensitive = false
		has = [
			{
				type  = "header"
				key   = "x-region"
				value = "uk"
			}
		]
	}
}
`, nameSuffix)
}

func testAccProjectRouteConfigWithoutPromo(nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-project-route-%s"
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
`, nameSuffix)
}

func testAccProjectRouteImportConfig(nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-project-route-import-%s"
}

resource "vercel_project_route" "example" {
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
`, nameSuffix)
}
