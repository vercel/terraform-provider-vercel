package vercel_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_EdgeConfigItemDataSource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEdgeConfigItemDataSourceConfig(name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.vercel_edge_config_item.test", "id"),
					resource.TestCheckResourceAttrSet("data.vercel_edge_config_item.test", "team_id"),
					resource.TestCheckResourceAttr("data.vercel_edge_config_item.test", "key", "foobar"),
					resource.TestCheckResourceAttr("data.vercel_edge_config_item.test", "value", "baz"),
				),
			},
			{
				Config:      testAccEdgeConfigItemDataSourceConfigNoItem(name, teamIDConfig()),
				ExpectError: regexp.MustCompile("not_found"),
			},
		},
	})
}

func testAccEdgeConfigItemDataSourceConfig(name, teamID string) string {
	return fmt.Sprintf(`
resource "vercel_edge_config" "test" {
    name         = "%[1]s"
    %[2]s
}

resource "vercel_edge_config_item" "test" {
    edge_config_id = vercel_edge_config.test.id
    key = "foobar"
    value = "baz"
    %[2]s
}

data "vercel_edge_config_item" "test" {
    id = vercel_edge_config_item.test.edge_config_id
    key = "foobar"
    %[2]s
}
`, name, teamID)
}

func testAccEdgeConfigItemDataSourceConfigNoItem(name, teamID string) string {
	return fmt.Sprintf(`
resource "vercel_edge_config" "test" {
    name         = "%[1]s"
    %[2]s
}

data "vercel_edge_config_item" "test" {
    id = vercel_edge_config.test.id
    key = "foobar"
    %[2]s
}
`, name, teamID)
}
