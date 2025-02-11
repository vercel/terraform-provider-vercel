package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testCheckIntegrationProjectAccessDestroyed(n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		allowed, err := testClient().GetIntegrationProjectAccess(context.TODO(), rs.Primary.Attributes["integration_id"], rs.Primary.Attributes["project_id"], teamID)
		if err != nil {
			return err
		}
		if allowed {
			return fmt.Errorf("expected project to not allow access to integration")
		}

		return nil
	}
}

func testCheckIntegrationProjectAccessExists(n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		allowed, err := testClient().GetIntegrationProjectAccess(context.TODO(), rs.Primary.Attributes["integration_id"], rs.Primary.Attributes["project_id"], teamID)
		if err != nil {
			return err
		}
		if !allowed {
			return fmt.Errorf("expected project to allow access to integration")
		}

		return nil
	}
}

func TestAcc_IntegrationProjectAccess(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckIntegrationProjectAccessDestroyed("vercel_integration_project_access.test_integration_access", testTeam()),
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationProjectAccess(name, teamIDConfig(), testExistingIntegration()),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckIntegrationProjectAccessExists("vercel_integration_project_access.test_integration_access", testTeam()),
					resource.TestCheckResourceAttr("vercel_integration_project_access.test_integration_access", "allowed", "true"),
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
		allowed        = true
    %[2]s
}
`, name, team, integration)
}
