package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v5/client"
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
				Config: cfg(testAccResourceEdgeConfigItem(name, "foobar", "baz")),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckEdgeConfigExists(testClient(t), testTeam(t), "vercel_edge_config.test_item"),
					resource.TestCheckResourceAttr("vercel_edge_config_item.test", "key", "foobar"),
					resource.TestCheckResourceAttr("vercel_edge_config_item.test", "value", "baz"),
				),
			},
			{
				// Changing the value is an in-place update (upsert), so the key is
				// never absent from the Edge Config between the old and new value.
				Config: cfg(testAccResourceEdgeConfigItem(name, "foobar", "qux")),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("vercel_edge_config_item.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_edge_config_item.test", "key", "foobar"),
					resource.TestCheckResourceAttr("vercel_edge_config_item.test", "value", "qux"),
				),
			},
			{
				// Changing the key still forces a replacement.
				Config: cfg(testAccResourceEdgeConfigItem(name, "foobar2", "qux")),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("vercel_edge_config_item.test", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_edge_config_item.test", "key", "foobar2"),
					resource.TestCheckResourceAttr("vercel_edge_config_item.test", "value", "qux"),
					testCheckEdgeConfigItemDeleted(testClient(t), "vercel_edge_config.test_item", "foobar", testTeam(t)),
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
					testCheckEdgeConfigItemDeleted(testClient(t), "vercel_edge_config.test_item", "foobar2", testTeam(t)),
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
				Config: cfg(testAccResourceEdgeConfigItemJSON(name, "true")),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckEdgeConfigExists(testClient(t), testTeam(t), "vercel_edge_config.test_item"),
					resource.TestCheckResourceAttr("vercel_edge_config_item.test_json", "key", "flags"),
					resource.TestCheckResourceAttr("vercel_edge_config_item.test_json", "value_json.featureA", "true"),
				),
			},
			{
				// Changing a structured value is also an in-place update.
				Config: cfg(testAccResourceEdgeConfigItemJSON(name, "false")),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("vercel_edge_config_item.test_json", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_edge_config_item.test_json", "key", "flags"),
					resource.TestCheckResourceAttr("vercel_edge_config_item.test_json", "value_json.featureA", "false"),
				),
			},
		},
	})
}

func testAccResourceEdgeConfigItem(name, key, value string) string {
	return fmt.Sprintf(`
resource "vercel_edge_config" "test_item" {
    name         = "%[1]s"
}

resource "vercel_edge_config_item" "test" {
    edge_config_id = vercel_edge_config.test_item.id
    key = "%[2]s"
    value = "%[3]s"
}
`, name, key, value)
}

func testAccResourceEdgeConfigItemJSON(name, featureA string) string {
	return fmt.Sprintf(`
resource "vercel_edge_config" "test_item" {
    name         = "%[1]s"
}

resource "vercel_edge_config_item" "test_json" {
    edge_config_id = vercel_edge_config.test_item.id
    key = "flags"
    value_json = {
      featureA = %[2]s
      nested = { a = 1, b = [1,2,3] }
    }
}
`, name, featureA)
}

func testAccResourceEdgeConfigItemDeleted(name string) string {
	return fmt.Sprintf(`
resource "vercel_edge_config" "test_item" {
    name         = "%[1]s"
}
`, name)
}
