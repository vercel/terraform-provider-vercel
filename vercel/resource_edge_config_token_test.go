package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v2/client"
)

func testCheckEdgeConfigTokenExists(teamID, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient().GetEdgeConfigToken(context.TODO(), client.EdgeConfigTokenRequest{
			Token:        rs.Primary.Attributes["token"],
			EdgeConfigID: rs.Primary.Attributes["edge_config_id"],
			TeamID:       teamID,
		})
		if err != nil {
			return fmt.Errorf("error getting %s/%s/%s: %w", teamID, rs.Primary.Attributes["edge_config_id"], rs.Primary.ID, err)
		}
		return err
	}
}

func testCheckEdgeConfigTokenDeleted(n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient().GetEdgeConfigToken(context.TODO(), client.EdgeConfigTokenRequest{
			Token:        rs.Primary.ID,
			EdgeConfigID: rs.Primary.Attributes["edge_config_id"],
			TeamID:       teamID,
		})
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("Unexpected error checking for deleted edge config token: %s", err)
		}

		return nil
	}
}

func TestAcc_EdgeConfigTokenResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckEdgeConfigTokenDeleted("vercel_edge_config_token.test", testTeam()),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceEdgeConfigToken(name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckEdgeConfigTokenExists(testTeam(), "vercel_edge_config_token.test"),
					resource.TestCheckResourceAttr("vercel_edge_config_token.test", "label", "test token"),
					resource.TestCheckResourceAttrSet("vercel_edge_config_token.test", "id"),
					resource.TestCheckResourceAttrSet("vercel_edge_config_token.test", "edge_config_id"),
					resource.TestCheckResourceAttrSet("vercel_edge_config_token.test", "connection_string"),
				),
			},
		},
	})
}

func testAccResourceEdgeConfigToken(name, team string) string {
	return fmt.Sprintf(`
resource "vercel_edge_config" "test" {
    name         = "%[1]s"
    %[2]s
}

resource "vercel_edge_config_token" "test" {
    label = "test token"
    edge_config_id = vercel_edge_config.test.id
    %[2]s
}
`, name, team)
}
