package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_NetworkDataSource(t *testing.T) {
	const networkID = "h8jgfvz9g75zpeq9"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccNetworkDataSourceByID(networkID)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_network.test", "id", networkID),
					resource.TestCheckResourceAttrSet("data.vercel_network.test", "name"),
					resource.TestCheckResourceAttrSet("data.vercel_network.test", "cidr"),
					resource.TestCheckResourceAttrSet("data.vercel_network.test", "region"),
					resource.TestCheckResourceAttrSet("data.vercel_network.test", "aws_account_id"),
					resource.TestCheckResourceAttrSet("data.vercel_network.test", "aws_region"),
					resource.TestCheckResourceAttrSet("data.vercel_network.test", "status"),
					resource.TestCheckResourceAttrSet("data.vercel_network.test", "team_id"),
				),
			},
		},
	})
}

func testAccNetworkDataSourceByID(id string) string {
	return fmt.Sprintf(`
data "vercel_network" "test" {
  id = "%s"
}
`, id)
}
