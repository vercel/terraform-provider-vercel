package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_EdgeConfigDataSource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEdgeConfigDataSourceConfig(name, teamIDConfig(t)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_edge_config.test", "name", name),
				),
			},
		},
	})
}

func testAccEdgeConfigDataSourceConfig(name, teamID string) string {
	return fmt.Sprintf(`
resource "vercel_edge_config" "test" {
    name         = "%[1]s"
    %[2]s
}

data "vercel_edge_config" "test" {
    id = vercel_edge_config.test.id
    %[2]s
}
`, name, teamID)
}
