package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_EdgeConfigTokenDataSource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEdgeConfigTokenDataSourceConfig(name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_edge_config_token.test", "label", "test-acc-token"),
					resource.TestCheckResourceAttrSet("data.vercel_edge_config_token.test", "edge_config_id"),
					resource.TestCheckResourceAttrSet("data.vercel_edge_config_token.test", "connection_string"),
					resource.TestCheckResourceAttrSet("data.vercel_edge_config_token.test", "id"),
					resource.TestCheckResourceAttrSet("data.vercel_edge_config_token.test", "token"),
				),
			},
		},
	})
}

func testAccEdgeConfigTokenDataSourceConfig(name, teamID string) string {
	return fmt.Sprintf(`
resource "vercel_edge_config" "test" {
    name = "%[1]s"
    %[2]s
}

resource "vercel_edge_config_token" "test" {
    label          = "test-acc-token"
    edge_config_id = vercel_edge_config.test.id
    %[2]s
}

data "vercel_edge_config_token" "test" {
    edge_config_id = vercel_edge_config.test.id
    token          = vercel_edge_config_token.test.token
    %[2]s
}
`, name, teamID)
}
