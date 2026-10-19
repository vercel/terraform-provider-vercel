package vercel_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestAcc_FeatureFlagConnectionResource(t *testing.T) {
	projectSuffix := strings.ToLower(acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckFeatureFlagConnectionDeleted(testClient(t), "vercel_feature_flag_connection.test"),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccFeatureFlagConnectionResourceConfig(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckFeatureFlagConnectionExists(testClient(t), "vercel_feature_flag_connection.test"),
					resource.TestCheckResourceAttrPair("vercel_feature_flag_connection.test", "project_id", "vercel_project.source", "id"),
					resource.TestCheckResourceAttrPair("vercel_feature_flag_connection.test", "consumer_project_id", "vercel_project.consumer", "id"),
				),
			},
			{
				ResourceName:      "vercel_feature_flag_connection.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getFeatureFlagConnectionImportID("vercel_feature_flag_connection.test"),
			},
		},
	})
}

func getFeatureFlagConnectionImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}

		return fmt.Sprintf(
			"%s/%s/%s",
			rs.Primary.Attributes["team_id"],
			rs.Primary.Attributes["project_id"],
			rs.Primary.Attributes["consumer_project_id"],
		), nil
	}
}

func testCheckFeatureFlagConnectionExists(testClient *client.Client, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		connections, err := testClient.ListFeatureFlagConnections(context.TODO(), client.ListFeatureFlagConnectionsRequest{
			ProjectID: rs.Primary.Attributes["project_id"],
			TeamID:    rs.Primary.Attributes["team_id"],
		})
		if err != nil {
			return err
		}

		for _, connection := range connections {
			if connection.ConsumerProjectID == rs.Primary.Attributes["consumer_project_id"] {
				return nil
			}
		}

		return fmt.Errorf("feature flag connection %s not found", rs.Primary.ID)
	}
}

func testCheckFeatureFlagConnectionDeleted(testClient *client.Client, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return nil
		}

		connections, err := testClient.ListFeatureFlagConnections(context.TODO(), client.ListFeatureFlagConnectionsRequest{
			ProjectID: rs.Primary.Attributes["project_id"],
			TeamID:    rs.Primary.Attributes["team_id"],
		})
		if err != nil {
			if client.NotFound(err) || strings.Contains(err.Error(), "project_not_found") {
				return nil
			}
			return err
		}

		for _, connection := range connections {
			if connection.ConsumerProjectID == rs.Primary.Attributes["consumer_project_id"] {
				return fmt.Errorf("feature flag connection %s still exists", rs.Primary.ID)
			}
		}

		return nil
	}
}

func testAccFeatureFlagConnectionResourceConfig(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "source" {
  name = "test-acc-flag-connection-source-%[1]s"
}

resource "vercel_project" "consumer" {
  name = "test-acc-flag-connection-consumer-%[1]s"
}

resource "vercel_feature_flag_definition" "bootstrap" {
  project_id = vercel_project.source.id
  key        = "bootstrap-%[1]s"
  kind       = "boolean"
  variant = [
    {
      id         = "off"
      value_bool = false
    },
    {
      id         = "on"
      value_bool = true
    },
  ]
}

resource "vercel_feature_flag_connection" "test" {
  project_id          = vercel_project.source.id
  consumer_project_id = vercel_project.consumer.id

  depends_on = [vercel_feature_flag_definition.bootstrap]
}
`, projectSuffix)
}
