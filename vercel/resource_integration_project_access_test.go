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

func testCheckIntegrationProjectAccessDestroyed(testClient *client.Client, n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		ipa, err := testClient.GetIntegrationProjectAccess(context.TODO(), rs.Primary.Attributes["integration_id"], rs.Primary.Attributes["project_id"], teamID)
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
		CheckDestroy:             testCheckIntegrationProjectAccessDestroyed(testClient(t), "vercel_integration_project_access.test_integration_access", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationProjectAccess(name, teamIDConfig(t), testExistingIntegration(t)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckIntegrationProjectAccessExists(testClient(t), "vercel_integration_project_access.test_integration_access", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_integration_project_access.test_integration_access", "team_id", testTeam(t)),
				),
			},
		},
	})
}

func TestAcc_IntegrationProjectAccessWithoutExplicitTeam(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckIntegrationProjectAccessDestroyed(testClient(t), "vercel_integration_project_access.test_integration_access", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationProjectAccessUsingProvider(name, testTeam(t), testExistingIntegration(t)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckIntegrationProjectAccessExists(testClient(t), "vercel_integration_project_access.test_integration_access", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_integration_project_access.test_integration_access", "team_id", testTeam(t)),
				),
			},
		},
	})
}

func testAccIntegrationProjectAccess(name, team, integration string) string {
	return fmt.Sprintf(`
data "vercel_endpoint_verification" "test" {
    %[2]s
}

resource "vercel_project" "test" {
    name = "test-acc-%[1]s"
    %[2]s
}

resource "vercel_integration_project_access" "test_integration_access" {
    integration_id = "%[3]s"
    project_id     = vercel_project.test.id
    %[2]s
}
`, name, team, integration)
}

func testAccIntegrationProjectAccessUsingProvider(name, team, integration string) string {
	//lintignore:AT004
	return fmt.Sprintf(`
provider "vercel" {
	team = "%[2]s"
}

data "vercel_endpoint_verification" "test" {
}

resource "vercel_project" "test" {
    name = "test-acc-%[1]s"
}

resource "vercel_integration_project_access" "test_integration_access" {
    integration_id = "%[3]s"
    project_id     = vercel_project.test.id
}
`, name, team, integration)
}
