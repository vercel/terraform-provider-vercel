package vercel_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v2/client"
)

func testCheckEdgeConfigSchemaExists(teamID, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient().GetEdgeConfigSchema(context.TODO(), rs.Primary.ID, teamID)
		if err != nil {
			return fmt.Errorf("error getting %s/%s: %w", teamID, rs.Primary.ID, err)
		}
		return err
	}
}

func testCheckEdgeConfigSchemaDeleted(n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient().GetEdgeConfigSchema(context.TODO(), rs.Primary.ID, teamID)
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("Unexpected error checking for deleted edge config schema: %s", err)
		}

		return nil
	}
}

func TestAcc_EdgeConfigSchemaResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckEdgeConfigSchemaDeleted("vercel_edge_config_schema.test", testTeam()),
		Steps: []resource.TestStep{
			{
				Config: `
                resource "vercel_edge_config_schema" "test" {
                    id = "shouldnt-matter"
                    definition = <<EOF
                    {
                        invalidjson: "foo"
                    }
                    EOF
                }
                `,
				ExpectError: regexp.MustCompile("Value must be a valid JSON document"),
			},
			{
				Config: testAccResourceEdgeConfigSchema(name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckEdgeConfigSchemaExists(testTeam(), "vercel_edge_config_schema.test"),
					resource.TestCheckResourceAttrSet("vercel_edge_config_schema.test", "id"),
					resource.TestCheckResourceAttrSet("vercel_edge_config_schema.test", "definition"),
				),
			},
			{
				Config: testAccResourceEdgeConfigSchemaUpdated(name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckEdgeConfigSchemaExists(testTeam(), "vercel_edge_config_schema.test"),
					resource.TestCheckResourceAttrSet("vercel_edge_config_schema.test", "id"),
					resource.TestCheckResourceAttrSet("vercel_edge_config_schema.test", "definition"),
				),
			},
		},
	})
}

func testAccResourceEdgeConfigSchema(name, team string) string {
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
`, name, team)
}

func testAccResourceEdgeConfigSchemaUpdated(name, team string) string {
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
    },
    "name": {
      "description": "The persons name",
      "type": "string"
    }
  }
}
EOF
    %[2]s
}
`, name, team)
}
