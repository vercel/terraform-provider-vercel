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
					resource.TestCheckResourceAttr(resourceName, "list.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "map.%", "2"),
					resource.TestCheckResourceAttrSet(resourceName, "list.0.id"),
					resource.TestCheckResourceAttrSet(resourceName, "list.0.name"),
					resource.TestCheckResourceAttrSet(resourceName, "list.1.id"),
					resource.TestCheckResourceAttrSet(resourceName, "list.1.name"),
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
