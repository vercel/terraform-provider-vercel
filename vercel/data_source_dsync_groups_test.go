package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_DsyncGroupsDataSource(t *testing.T) {
	resourceName := "data.vercel_dsync_groups.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVercelDsyncGroupsDataSource(testTeam(t)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "groups.#"),
					resource.TestCheckResourceAttrSet(resourceName, "groups.0.id"),
					resource.TestCheckResourceAttrSet(resourceName, "groups.0.name"),
					resource.TestCheckResourceAttrSet(resourceName, "groups.1.id"),
					resource.TestCheckResourceAttrSet(resourceName, "groups.1.name"),
				),
			},
		},
	})
}

func testAccVercelDsyncGroupsDataSource(teamID string) string {
	return fmt.Sprintf(`
data "vercel_dsync_groups" "test" {
  team_id = "%s"
}
`, teamID)
}
