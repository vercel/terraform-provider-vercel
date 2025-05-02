package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_TeamConfigDataSource(t *testing.T) {
	resourceName := "data.vercel_team_config.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVercelTeamConfigDataSource(testTeam(t)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "name"),
					resource.TestCheckResourceAttrSet(resourceName, "slug"),
					resource.TestCheckResourceAttrSet(resourceName, "description"),
					resource.TestCheckResourceAttrSet(resourceName, "sensitive_environment_variable_policy"),
					resource.TestCheckResourceAttrSet(resourceName, "remote_caching.enabled"),
				),
			},
		},
	})
}

func testAccVercelTeamConfigDataSource(teamID string) string {
	return fmt.Sprintf(`
data "vercel_team_config" "test" {
  id   = "%s"
}
`, teamID)
}
