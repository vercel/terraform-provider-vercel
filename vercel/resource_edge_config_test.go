package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

func testCheckEdgeConfigExists(testClient *client.Client, teamID, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetEdgeConfig(context.TODO(), rs.Primary.ID, teamID)
		return err
	}
}

func testCheckEdgeConfigDeleted(testClient *client.Client, n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetEdgeConfig(context.TODO(), rs.Primary.ID, teamID)
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("Unexpected error checking for deleted edge config: %s", err)
		}

		return nil
	}
}

func TestAcc_EdgeConfigResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckEdgeConfigDeleted(testClient(t), "vercel_edge_config.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccResourceEdgeConfig(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckEdgeConfigExists(testClient(t), testTeam(t), "vercel_edge_config.test"),
					resource.TestCheckResourceAttr("vercel_edge_config.test", "name", name),
					resource.TestCheckResourceAttrSet("vercel_edge_config.test", "id"),
				),
			},
			{
				Config: cfg(testAccResourceEdgeConfigUpdated(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckEdgeConfigExists(testClient(t), testTeam(t), "vercel_edge_config.test"),
					resource.TestCheckResourceAttr("vercel_edge_config.test", "name", fmt.Sprintf("%s-updated", name)),
					resource.TestCheckResourceAttrSet("vercel_edge_config.test", "id"),
				),
			},
		},
	})
}

func testAccResourceEdgeConfig(name string) string {
	return fmt.Sprintf(`
resource "vercel_edge_config" "test" {
    name         = "%[1]s"
}
`, name)
}

func testAccResourceEdgeConfigUpdated(name string) string {
	return fmt.Sprintf(`
resource "vercel_edge_config" "test" {
    name         = "%[1]s-updated"
}
`, name)
}
