package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/client"
)

func testCheckEdgeConfigExists(teamID, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient().GetEdgeConfig(context.TODO(), rs.Primary.ID, teamID)
		return err
	}
}

func testCheckEdgeConfigDeleted(n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient().GetEdgeConfig(context.TODO(), rs.Primary.ID, teamID)
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("Unexpected error checking for deleted project: %s", err)
		}

		return nil
	}
}

func TestAcc_EdgeConfigResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckEdgeConfigDeleted("vercel_edge_config.test", testTeam()),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceEdgeConfig(name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckEdgeConfigExists(testTeam(), "vercel_edge_config.test"),
					resource.TestCheckResourceAttr("vercel_edge_config.test", "name", name),
					resource.TestCheckResourceAttrSet("vercel_edge_config.test", "id"),
				),
			},
			{
				Config: testAccResourceEdgeConfigUpdated(name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckEdgeConfigExists(testTeam(), "vercel_edge_config.test"),
					resource.TestCheckResourceAttr("vercel_edge_config.test", "name", fmt.Sprintf("%s-updated", name)),
					resource.TestCheckResourceAttrSet("vercel_edge_config.test", "id"),
				),
			},
		},
	})
}

func testAccResourceEdgeConfig(name, team string) string {
	return fmt.Sprintf(`
resource "vercel_edge_config" "test" {
    name         = "%[1]s"
    %[2]s
}
`, name, team)
}

func testAccResourceEdgeConfigUpdated(name, team string) string {
	return fmt.Sprintf(`
resource "vercel_edge_config" "test" {
    name         = "%[1]s-updated"
    %[2]s
}
`, name, team)
}
