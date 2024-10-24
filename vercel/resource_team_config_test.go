package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_TeamConfig(t *testing.T) {
	resourceName := "vercel_team_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVercelTeamConfigBasic(testTeam()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "vercel-terraform-test-ci"),
					resource.TestCheckResourceAttr(resourceName, "slug", "vercel-terraform-test-ci"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			{
				Config: testAccVercelTeamConfigUpdated(testTeam()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "vercel-terraform-test-ci-"),
					resource.TestCheckResourceAttr(resourceName, "description", "Vercel Terraform Testing"),
					resource.TestCheckResourceAttr(resourceName, "sensitive_environment_variable_policy", "off"),
                    resource.Test
				),
			},
		},
	})
}

func testAccVercelTeamConfigBasic(teamID string) string {
	return fmt.Sprintf(`
resource "vercel_team_config" "test" {
  id   = "%s" // Replace with a valid team ID
  name = "vercel-terraform-test-ci"
  slug = "vercel-terraform-test-ci"
}
`, teamID)
}

func testAccVercelTeamConfigUpdated(teamID string) string {
	return fmt.Sprintf(`
resource "vercel_team_config" "test" {
  id                                    = "%s"
  name                                  = "vercel-terraform-test-ci-"
  slug                                  = "vercel-terraform-test-ci-"
  description                           = "Vercel Terraform Testing"
  sensitive_environment_variable_policy = "off"
  remote_caching {
    enabled = true
  }
  enable_preview_feedback = "off"
  enable_production_feedback = "off"
  hide_ip_addresses = true
  hide_ip_addresses_in_log_drains = true
}
`, teamID)
}
