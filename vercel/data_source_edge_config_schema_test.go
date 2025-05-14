package vercel_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_EdgeConfigSchemaDataSource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccEdgeConfigSchemaDataSourceConfig(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.vercel_edge_config_schema.test", "id"),
					resource.TestCheckResourceAttrSet("data.vercel_edge_config_schema.test", "team_id"),
					resource.TestCheckResourceAttrSet("data.vercel_edge_config_schema.test", "definition"),
				),
			},
			{
				Config:      cfg(testAccEdgeConfigSchemaDataSourceConfigNoSchema(name)),
				ExpectError: regexp.MustCompile("not_found"),
			},
		},
	})
}

func testAccEdgeConfigSchemaDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "vercel_edge_config" "test" {
    name         = "%[1]s"
}

resource "vercel_edge_config_schema" "test" {
    id = vercel_edge_config.test.id
    definition = <<EOF
{
  "title": "Greeting",
  "type": "object",
  "properties": {
    "greeting": {
      "description": "A friendly greeting",
      "type": "string"
    }
  }
}
EOF
}

data "vercel_edge_config_schema" "test" {
    id = vercel_edge_config_schema.test.id
}
`, name)
}

func testAccEdgeConfigSchemaDataSourceConfigNoSchema(name string) string {
	return fmt.Sprintf(`
resource "vercel_edge_config" "test" {
    name         = "%[1]s"
}

data "vercel_edge_config_schema" "test" {
    id = vercel_edge_config.test.id
}
`, name)
}
