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

func TestAcc_AccessGroupProjectResource(t *testing.T) {
	name := acctest.RandString(16)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccAccessGroupProjectDoesNotExist(testClient(t), testTeam(t), "vercel_access_group_project.test"),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccResourceAccessGroupProject(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAccessGroupProjectExists(testClient(t), testTeam(t), "vercel_access_group_project.test"),
					resource.TestCheckResourceAttrSet("vercel_access_group_project.test", "access_group_id"),
					resource.TestCheckResourceAttrSet("vercel_access_group_project.test", "project_id"),
					resource.TestCheckResourceAttr("vercel_access_group_project.test", "role", "ADMIN"),
				),
			},
			{
				ResourceName:      "vercel_access_group_project.test",
				ImportState:       true,
				ImportStateIdFunc: getAccessGroupProjectImportID("vercel_access_group_project.test"),
			},
			{
				Config: cfg(testAccResourceAccessGroupProjectUpdated(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAccessGroupProjectExists(testClient(t), testTeam(t), "vercel_access_group_project.test"),
					resource.TestCheckResourceAttrSet("vercel_access_group_project.test", "project_id"),
					resource.TestCheckResourceAttrSet("vercel_access_group_project.test", "access_group_id"),
					resource.TestCheckResourceAttr("vercel_access_group_project.test", "role", "PROJECT_DEVELOPER"),
				),
			},
		},
	})
}

func getAccessGroupProjectImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}

		return fmt.Sprintf(
			"%s/%s/%s",
			rs.Primary.Attributes["team_id"],
			rs.Primary.Attributes["access_group_id"],
			rs.Primary.Attributes["project_id"],
		), nil
	}
}

func testCheckAccessGroupProjectExists(testClient *client.Client, teamID string, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		_, err := testClient.GetAccessGroupProject(context.TODO(), client.GetAccessGroupProjectRequest{
			TeamID:        teamID,
			AccessGroupID: rs.Primary.Attributes["access_group_id"],
			ProjectID:     rs.Primary.Attributes["project_id"],
		})
		return err
	}
}

func testAccAccessGroupProjectDoesNotExist(testClient *client.Client, teamID string, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		_, err := testClient.GetAccessGroupProject(context.TODO(), client.GetAccessGroupProjectRequest{
			TeamID:        teamID,
			AccessGroupID: rs.Primary.Attributes["access_group_id"],
			ProjectID:     rs.Primary.Attributes["project_id"],
		})
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("Unexpected error checking for deleted access group: %s", err)
		}

		return nil
	}
}

func testAccResourceAccessGroupProject(name string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-%[1]s"
}

resource "vercel_access_group" "test" {
	name = "test-acc-%[1]s"
}

resource "vercel_access_group_project" "test" {
	project_id = vercel_project.test.id
	access_group_id = vercel_access_group.test.id
	role = "ADMIN"
}
`, name)
}

func testAccResourceAccessGroupProjectUpdated(name string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-%[1]s"
}

resource "vercel_access_group" "test" {
	name = "test-acc-%[1]s"
}

resource "vercel_access_group_project" "test" {
	project_id = vercel_project.test.id
	access_group_id = vercel_access_group.test.id
	role = "PROJECT_DEVELOPER"
}
`, name)
}
