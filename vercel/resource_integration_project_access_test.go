package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func testCheckIntegrationProjectAccessRevoked(testClient *client.Client, n, integrationID, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		ipa, err := testClient.GetIntegrationProjectAccess(context.TODO(), integrationID, rs.Primary.ID, teamID)
		if err != nil {
			return err
		}
		if ipa.Allowed {
			return fmt.Errorf("expected project to not allow access to integration")
		}

		return nil
	}
}

func testCheckIntegrationProjectAccessExists(testClient *client.Client, n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		ipa, err := testClient.GetIntegrationProjectAccess(context.TODO(), rs.Primary.Attributes["integration_id"], rs.Primary.Attributes["project_id"], teamID)
		if err != nil {
			return err
		}
		if !ipa.Allowed {
			return fmt.Errorf("expected project to allow access to integration")
		}

		return nil
	}
}

func TestAcc_IntegrationProjectAccess(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccIntegrationProjectAccess(name, testExistingIntegration(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckIntegrationProjectAccessExists(testClient(t), "vercel_integration_project_access.test_integration_access", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_integration_project_access.test_integration_access", "team_id", testTeam(t)),
				),
			},
			{
				// Verify revocation before deleting the project: the access endpoint
				// returns 404 once the project no longer exists.
				Config: cfg(testAccIntegrationProjectAccessProject(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists(testClient(t), "vercel_project.test", testTeam(t)),
					testCheckIntegrationProjectAccessRevoked(testClient(t), "vercel_project.test", testExistingIntegration(t), testTeam(t)),
				),
			},
		},
	})
}

func testAccIntegrationProjectAccess(name, integration string) string {
	return testAccIntegrationProjectAccessProject(name) + fmt.Sprintf(`
resource "vercel_integration_project_access" "test_integration_access" {
    integration_id = "%s"
    project_id     = vercel_project.test.id
}
`, integration)
}

func testAccIntegrationProjectAccessProject(name string) string {
	return fmt.Sprintf(`
data "vercel_endpoint_verification" "test" {
}

resource "vercel_project" "test" {
    name = "test-acc-%[1]s"
}

`, name)
}
