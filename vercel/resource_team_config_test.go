package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_TeamConfig(t *testing.T) {
	resourceName := "vercel_team_config.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVercelTeamConfigBasic(testTeam(t)),
				// Added since vercel_team_config schema version upgraded
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Vercel Terraform Testing"),
					resource.TestCheckResourceAttr(resourceName, "slug", "terraform-testing-vtest314"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			{
				Config: testAccVercelTeamConfigUpdated(testTeam(t)),
				// Added since vercel_team_config schema version upgraded
				ExpectNonEmptyPlan: true,
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
				),
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
}
`, teamID)
}
