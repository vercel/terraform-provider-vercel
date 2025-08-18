package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAcc_TeamConfig(t *testing.T) {
	resourceName := "vercel_team_config.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccVercelTeamConfigBasic(testTeam(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Vercel Terraform Testing"),
					resource.TestCheckResourceAttr(resourceName, "slug", "terraform-testing-vtest314"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			{
				Config: cfg(testAccVercelTeamConfigUpdated(testTeam(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", "Vercel Terraform Testing o_o"),
					resource.TestCheckResourceAttr(resourceName, "slug", "terraform-testing-vtest314"),
					resource.TestCheckResourceAttr(resourceName, "description", "Vercel Terraform Testing"),
					resource.TestCheckResourceAttr(resourceName, "sensitive_environment_variable_policy", "off"),
					resource.TestCheckResourceAttr(resourceName, "remote_caching.enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "enable_preview_feedback", "off"),
					resource.TestCheckResourceAttr(resourceName, "enable_production_feedback", "off"),
					resource.TestCheckResourceAttr(resourceName, "hide_ip_addresses", "true"),
					resource.TestCheckResourceAttr(resourceName, "hide_ip_addresses_in_log_drains", "true"),
					resource.TestCheckResourceAttr(resourceName, "on_demand_concurrent_builds", "true"),
				),
			},
			{
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"avatar"},
				ImportStateIdFunc:       getTeamConfigImportID(resourceName),
				ResourceName:            resourceName,
			},
		},
	})
}

func testAccVercelTeamConfigBasic(teamID string) string {
	return fmt.Sprintf(`
resource "vercel_team_config" "test" {
  id   = "%s"
  name = "Vercel Terraform Testing"
}
`, teamID)
}

func testAccVercelTeamConfigUpdated(teamID string) string {
	return fmt.Sprintf(`
data "vercel_file" "test" {
    path = "examples/avatar.png"
}

resource "vercel_team_config" "test" {
  id                                    = "%s"
  avatar                                =  data.vercel_file.test.file
  name                                  = "Vercel Terraform Testing o_o"
  slug                                  = "terraform-testing-vtest314"
  description                           = "Vercel Terraform Testing"
  sensitive_environment_variable_policy = "off"
  remote_caching = {
    enabled = true
  }
  enable_preview_feedback = "off"
  enable_production_feedback = "off"
  hide_ip_addresses = true
  hide_ip_addresses_in_log_drains = true
  on_demand_concurrent_builds = true
}
`, teamID)
}

func getTeamConfigImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}
		return rs.Primary.Attributes["id"], nil
	}
}
