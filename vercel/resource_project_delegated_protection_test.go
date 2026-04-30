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

func getProjectDelegatedProtectionImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}

		teamID := rs.Primary.Attributes["team_id"]
		if teamID == "" {
			return rs.Primary.Attributes["project_id"], nil
		}

		return fmt.Sprintf("%s/%s", teamID, rs.Primary.Attributes["project_id"]), nil
	}
}

func testAccProjectDelegatedProtectionExists(testClient *client.Client, n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		project, err := testClient.GetProject(context.TODO(), rs.Primary.Attributes["project_id"], teamID)
		if err != nil {
			return err
		}
		if project.DelegatedProtection == nil {
			return fmt.Errorf("delegated protection is not enabled")
		}

		return nil
	}
}

func TestAcc_ProjectDelegatedProtection(t *testing.T) {
	nameSuffix := acctest.RandString(8)
	resourceName := "vercel_project_delegated_protection.example"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy(testClient(t), "vercel_project.example", testTeam(t)),
		),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectDelegatedProtectionConfig(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectDelegatedProtectionExists(testClient(t), resourceName, testTeam(t)),
					resource.TestCheckResourceAttr(resourceName, "client_id", "client-id-initial"),
					resource.TestCheckResourceAttr(resourceName, "issuer", "https://vercel.com"),
					resource.TestCheckResourceAttr(resourceName, "deployment_type", "standard_protection_new"),
					resource.TestCheckResourceAttr(resourceName, "cookie_name", "_vercel_delegated_custom"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateIdFunc:       getProjectDelegatedProtectionImportID(resourceName),
				ImportStateVerifyIgnore: []string{"client_secret"},
			},
			{
				Config: cfg(testAccProjectDelegatedProtectionConfigUpdated(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectDelegatedProtectionExists(testClient(t), resourceName, testTeam(t)),
					resource.TestCheckResourceAttr(resourceName, "client_id", "client-id-updated"),
					resource.TestCheckResourceAttr(resourceName, "issuer", "https://vercel.com"),
					resource.TestCheckResourceAttr(resourceName, "deployment_type", "standard_protection"),
					resource.TestCheckNoResourceAttr(resourceName, "cookie_name"),
				),
			},
		},
	})
}

func testAccProjectDelegatedProtectionConfig(nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-project-delegated-protection-%s"
	skew_protection = "12 hours"
}

resource "vercel_project_delegated_protection" "example" {
	project_id      = vercel_project.example.id
	client_id       = "client-id-initial"
	client_secret   = "client-secret-initial"
	cookie_name     = "_vercel_delegated_custom"
	deployment_type = "standard_protection_new"
	issuer          = "https://vercel.com"
}
`, nameSuffix)
}

func testAccProjectDelegatedProtectionConfigUpdated(nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-project-delegated-protection-%s"
	skew_protection = "12 hours"
}

resource "vercel_project_delegated_protection" "example" {
	project_id      = vercel_project.example.id
	client_id       = "client-id-updated"
	client_secret   = "client-secret-updated"
	deployment_type = "standard_protection"
	issuer          = "https://vercel.com"
}
`, nameSuffix)
}
