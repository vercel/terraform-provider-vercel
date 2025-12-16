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

func testCheckNetworkExists(testClient *client.Client, teamID, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.ReadNetwork(context.TODO(), client.ReadNetworkRequest{
			NetworkID: rs.Primary.ID,
			TeamID:    teamID,
		})
		return err
	}
}

func testCheckNetworkDeleted(testClient *client.Client, n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.ReadNetwork(context.TODO(), client.ReadNetworkRequest{
			NetworkID: rs.Primary.ID,
			TeamID:    teamID,
		})
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("Unexpected error checking for deleted network: %s", err)
		}

		return nil
	}
}

func TestAcc_NetworkResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckNetworkDeleted(testClient(t), "vercel_network.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccResourceNetwork(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckNetworkExists(testClient(t), testTeam(t), "vercel_network.test"),
					resource.TestCheckResourceAttr("vercel_network.test", "name", name),
					resource.TestCheckResourceAttr("vercel_network.test", "cidr", "10.0.0.0/16"),
					resource.TestCheckResourceAttr("vercel_network.test", "region", "iad1"),
					resource.TestCheckResourceAttrSet("vercel_network.test", "id"),
					resource.TestCheckResourceAttrSet("vercel_network.test", "team_id"),
					resource.TestCheckResourceAttrSet("vercel_network.test", "aws_account_id"),
					resource.TestCheckResourceAttrSet("vercel_network.test", "aws_region"),
					resource.TestCheckResourceAttrSet("vercel_network.test", "status"),
					resource.TestCheckResourceAttrSet("vercel_network.test", "vpc_id"),
				),
			},
			{
				ResourceName:      "vercel_network.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getNetworkImportID("vercel_network.test"),
			},
			{
				Config: cfg(testAccResourceNetworkUpdated(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckNetworkExists(testClient(t), testTeam(t), "vercel_network.test"),
					resource.TestCheckResourceAttr("vercel_network.test", "name", fmt.Sprintf("%s-updated", name)),
					resource.TestCheckResourceAttr("vercel_network.test", "cidr", "10.0.0.0/16"),
					resource.TestCheckResourceAttr("vercel_network.test", "region", "iad1"),
					resource.TestCheckResourceAttrSet("vercel_network.test", "id"),
					resource.TestCheckResourceAttrSet("vercel_network.test", "team_id"),
				),
			},
		},
	})
}

func getNetworkImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return "", fmt.Errorf("no ID is set")
		}

		teamID := rs.Primary.Attributes["team_id"]
		if teamID == "" {
			return rs.Primary.ID, nil
		}

		return fmt.Sprintf("%s/%s", teamID, rs.Primary.ID), nil
	}
}

func testAccResourceNetwork(name string) string {
	return fmt.Sprintf(`
resource "vercel_network" "test" {
    name   = "%[1]s"
    cidr   = "10.0.0.0/16"
    region = "iad1"
}
`, name)
}

func testAccResourceNetworkUpdated(name string) string {
	return fmt.Sprintf(`
resource "vercel_network" "test" {
    name   = "%[1]s-updated"
    cidr   = "10.0.0.0/16"
    region = "iad1"
}
`, name)
}
