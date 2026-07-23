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

func testCheckVCRRepositoryPermissionExists(testClient *client.Client, teamID string, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		projectID := rs.Primary.Attributes["project_id"]
		repository := rs.Primary.Attributes["repository"]
		grantedTeamID := rs.Primary.Attributes["granted_team_id"]

		_, err := testClient.GetVCRRepositoryPermission(context.TODO(), client.GetVCRRepositoryPermissionRequest{
			TeamID:        teamID,
			ProjectID:     projectID,
			IDOrName:      repository,
			GrantedTeamID: grantedTeamID,
		})
		if client.NotFound(err) {
			return fmt.Errorf("test failed because the vcr repository permission %s %s %s %s - %s could not be found", teamID, projectID, repository, grantedTeamID, rs.Primary.ID)
		}
		return err
	}
}

func TestAcc_VCRRepositoryPermissionResource(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccVCRRepositoryPermission(projectSuffix, testTeam(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckVCRRepositoryPermissionExists(testClient(t), testTeam(t), "vercel_vcr_repository_permission.test"),
					resource.TestCheckResourceAttrSet("vercel_vcr_repository_permission.test", "id"),
					resource.TestCheckResourceAttrSet("vercel_vcr_repository_permission.test", "repository_id"),
					resource.TestCheckResourceAttr("vercel_vcr_repository_permission.test", "granted_team_id", testTeam(t)),
					resource.TestCheckResourceAttrSet("vercel_vcr_repository_permission.test", "granted_team_slug"),
				),
			},
			{
				ResourceName:      "vercel_vcr_repository_permission.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getVCRRepositoryPermissionImportID("vercel_vcr_repository_permission.test"),
			},
		},
	})
}

func getVCRRepositoryPermissionImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return "", fmt.Errorf("no ID is set")
		}

		return fmt.Sprintf("%s/%s/%s/%s",
			rs.Primary.Attributes["team_id"],
			rs.Primary.Attributes["project_id"],
			rs.Primary.Attributes["repository"],
			rs.Primary.Attributes["granted_team_id"],
		), nil
	}
}

func testAccVCRRepositoryPermission(projectSuffix, grantedTeamID string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-vcr-perm-%[1]s"
}

resource "vercel_vcr_repository" "test" {
  project_id = vercel_project.test.id
  name       = "test-acc-%[1]s"
}

resource "vercel_vcr_repository_permission" "test" {
  project_id      = vercel_project.test.id
  repository      = vercel_vcr_repository.test.name
  granted_team_id = "%[2]s"
}
`, projectSuffix, grantedTeamID)
}
