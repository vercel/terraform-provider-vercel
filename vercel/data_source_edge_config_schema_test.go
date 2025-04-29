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
				Config: testAccEdgeConfigSchemaDataSourceConfig(name, teamIDConfig(t)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.vercel_edge_config_schema.test", "id"),
					resource.TestCheckResourceAttrSet("data.vercel_edge_config_schema.test", "team_id"),
					resource.TestCheckResourceAttrSet("data.vercel_edge_config_schema.test", "definition"),
				),
			},
			{
				Config:      testAccEdgeConfigSchemaDataSourceConfigNoSchema(name, teamIDConfig(t)),
				ExpectError: regexp.MustCompile("not_found"),
			},
		},
	})
}

func testAccEdgeConfigSchemaDataSourceConfig(name, teamID string) string {
	return fmt.Sprintf(`
resource "vercel_edge_config" "test" {
    name         = "%[1]s"
    %[2]s
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
    %[2]s
}

data "vercel_edge_config_schema" "test" {
    id = vercel_edge_config_schema.test.id
    %[2]s
}
`, name, teamID)
}

func testAccEdgeConfigSchemaDataSourceConfigNoSchema(name, teamID string) string {
	return fmt.Sprintf(`
resource "vercel_edge_config" "test" {
    name         = "%[1]s"
    %[2]s
}

data "vercel_edge_config_schema" "test" {
    id = vercel_edge_config.test.id
    %[2]s
}
`, name, teamID)
}
