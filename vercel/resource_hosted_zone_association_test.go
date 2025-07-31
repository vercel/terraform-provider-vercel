package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

func TestAcc_HostedZoneAssociationResource(t *testing.T) {
	configurationID := acctest.RandString(16)
	hostedZoneID := "Z" + acctest.RandString(13) // AWS Hosted Zone IDs start with Z
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testCheckHostedZoneAssociationDoesNotExist(testClient(t), testTeam(t), "vercel_hosted_zone_association.test"),
		),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccResourceHostedZoneAssociation(configurationID, hostedZoneID)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckHostedZoneAssociationExists(testClient(t), testTeam(t), "vercel_hosted_zone_association.test"),
					resource.TestCheckResourceAttrSet("vercel_hosted_zone_association.test", "configuration_id"),
					resource.TestCheckResourceAttrSet("vercel_hosted_zone_association.test", "hosted_zone_id"),
					resource.TestCheckResourceAttr("vercel_hosted_zone_association.test", "configuration_id", fmt.Sprintf("test-acc-%s", configurationID)),
					resource.TestCheckResourceAttr("vercel_hosted_zone_association.test", "hosted_zone_id", hostedZoneID),
					// These fields are computed and may be empty if the follow-up read fails
					resource.TestCheckResourceAttrSet("vercel_hosted_zone_association.test", "hosted_zone_name"),
					resource.TestCheckResourceAttrSet("vercel_hosted_zone_association.test", "owner"),
				),
			},
			{
				ResourceName:      "vercel_hosted_zone_association.test",
				ImportState:       true,
				ImportStateIdFunc: getHostedZoneAssociationImportID("vercel_hosted_zone_association.test"),
			},
		},
	})
}

func getHostedZoneAssociationImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return "", fmt.Errorf("no ID is set")
		}

		teamID := rs.Primary.Attributes["team_id"]
		configurationID := rs.Primary.Attributes["configuration_id"]
		hostedZoneID := rs.Primary.Attributes["hosted_zone_id"]

		if teamID != "" {
			return fmt.Sprintf("%s/%s/%s", teamID, configurationID, hostedZoneID), nil
		}
		return fmt.Sprintf("%s/%s", configurationID, hostedZoneID), nil
	}
}

func testCheckHostedZoneAssociationExists(testClient *client.Client, teamID string, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetHostedZoneAssociation(context.TODO(), client.GetHostedZoneAssociationRequest{
			TeamID:          teamID,
			ConfigurationID: rs.Primary.Attributes["configuration_id"],
			HostedZoneID:    rs.Primary.Attributes["hosted_zone_id"],
		})
		return err
	}
}

func testCheckHostedZoneAssociationDoesNotExist(testClient *client.Client, teamID string, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetHostedZoneAssociation(context.TODO(), client.GetHostedZoneAssociationRequest{
			TeamID:          teamID,
			ConfigurationID: rs.Primary.Attributes["configuration_id"],
			HostedZoneID:    rs.Primary.Attributes["hosted_zone_id"],
		})
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("unexpected error checking for deleted hosted zone association: %s", err)
		}

		return nil
	}
}

func testAccResourceHostedZoneAssociation(configurationID, hostedZoneID string) string {
	return fmt.Sprintf(`
resource "vercel_hosted_zone_association" "test" {
  configuration_id = "test-acc-%[1]s"
  hosted_zone_id   = "%[2]s"
}
`, configurationID, hostedZoneID)
}
