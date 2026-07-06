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

func testCheckVCRRepositoryExists(testClient *client.Client, teamID string, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		projectID := rs.Primary.Attributes["project_id"]
		name := rs.Primary.Attributes["name"]

		_, err := testClient.GetVCRRepository(context.TODO(), client.GetVCRRepositoryRequest{
			TeamID:    teamID,
			ProjectID: projectID,
			IDOrName:  name,
		})
		if client.NotFound(err) {
			return fmt.Errorf("test failed because the vcr repository %s %s %s - %s could not be found", teamID, projectID, name, rs.Primary.ID)
		}
		return err
	}
}

func TestAcc_VCRRepositoryResource(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccVCRRepository(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckVCRRepositoryExists(testClient(t), testTeam(t), "vercel_vcr_repository.test"),
					resource.TestCheckResourceAttrSet("vercel_vcr_repository.test", "id"),
					resource.TestCheckResourceAttrSet("vercel_vcr_repository.test", "project_id"),
					resource.TestCheckResourceAttr("vercel_vcr_repository.test", "name", fmt.Sprintf("test-acc-%s", projectSuffix)),
					resource.TestCheckResourceAttrSet("vercel_vcr_repository.test", "url"),
				),
			},
			{
				ResourceName:      "vercel_vcr_repository.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getVCRRepositoryImportID("vercel_vcr_repository.test"),
			},
		},
	})
}

func getVCRRepositoryImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return "", fmt.Errorf("no ID is set")
		}

		return fmt.Sprintf("%s/%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.Attributes["project_id"], rs.Primary.Attributes["name"]), nil
	}
}

func testAccVCRRepository(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-vcr-repo-%[1]s"
}

resource "vercel_vcr_repository" "test" {
  project_id = vercel_project.test.id
  name       = "test-acc-%[1]s"
}
`, projectSuffix)
}
