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

func getEdgeConfigItemImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return "", fmt.Errorf("no ID is set")
		}

		return fmt.Sprintf("%s/%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.Attributes["edge_config_id"], rs.Primary.Attributes["key"]), nil
	}
}

func testCheckEdgeConfigItemDeleted(testClient *client.Client, n, key, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetEdgeConfigItem(context.TODO(), client.EdgeConfigItemRequest{
			TeamID:       teamID,
			EdgeConfigID: rs.Primary.ID,
			Key:          key,
		})
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("Unexpected error checking for deleted edge config item: %s", err)
		}

		return nil
	}
}

func TestAcc_EdgeConfigItemResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckEdgeConfigDeleted(testClient(t), "vercel_edge_config.test_item", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccResourceEdgeConfigItem(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckEdgeConfigExists(testClient(t), testTeam(t), "vercel_edge_config.test_item"),
					resource.TestCheckResourceAttr("vercel_edge_config_item.test", "key", "foobar"),
					resource.TestCheckResourceAttr("vercel_edge_config_item.test", "value", "baz"),
				),
			},
			{
				ResourceName:      "vercel_edge_config_item.test",
				ImportState:       true,
				ImportStateIdFunc: getEdgeConfigItemImportID("vercel_edge_config_item.test"),
			},
			{
				Config: cfg(testAccResourceEdgeConfigItemDeleted(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckEdgeConfigExists(testClient(t), testTeam(t), "vercel_edge_config.test_item"),
					testCheckEdgeConfigItemDeleted(testClient(t), "vercel_edge_config.test_item", "foobar", testTeam(t)),
				),
			},
		},
	})
}

func TestAcc_EdgeConfigItemResource_JSON(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckEdgeConfigDeleted(testClient(t), "vercel_edge_config.test_item", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccResourceEdgeConfigItemJSON(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckEdgeConfigExists(testClient(t), testTeam(t), "vercel_edge_config.test_item"),
					resource.TestCheckResourceAttr("vercel_edge_config_item.test_json", "key", "flags"),
				),
			},
		},
	})
}

func testAccResourceEdgeConfigItem(name string) string {
	return fmt.Sprintf(`
resource "vercel_edge_config" "test_item" {
    name         = "%[1]s"
}

resource "vercel_edge_config_item" "test" {
    edge_config_id = vercel_edge_config.test_item.id
    key = "foobar"
    value = "baz"
}
`, name)
}

func testAccResourceEdgeConfigItemJSON(name string) string {
	return fmt.Sprintf(`
resource "vercel_edge_config" "test_item" {
    name         = "%[1]s"
}

resource "vercel_edge_config_item" "test_json" {
    edge_config_id = vercel_edge_config.test_item.id
    key = "flags"
    value_json = {
      featureA = true
      nested = { a = 1, b = [1,2,3] }
    }
}
`, name)
}

func testAccResourceEdgeConfigItemDeleted(name string) string {
	return fmt.Sprintf(`
resource "vercel_edge_config" "test_item" {
    name         = "%[1]s"
}
`, name)
}
