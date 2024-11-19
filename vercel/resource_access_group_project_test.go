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

func TestAcc_AccessGroupProjectResource(t *testing.T) {
	name := acctest.RandString(16)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccAccessGroupProjectDoesNotExist("vercel_access_group_project.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceAccessGroupProject(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAccessGroupProjectExists("vercel_access_group_project.test"),
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
				Config: testAccResourceAccessGroupProjectUpdated(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAccessGroupProjectExists("vercel_access_group_project.test"),
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

func testCheckAccessGroupProjectExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		_, err := testClient().GetAccessGroupProject(context.TODO(), client.GetAccessGroupProjectRequest{
			TeamID:        testTeam(),
			AccessGroupID: rs.Primary.Attributes["access_group_id"],
			ProjectID:     rs.Primary.Attributes["project_id"],
		})
		return err
	}
}

func testAccAccessGroupProjectDoesNotExist(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		_, err := testClient().GetAccessGroupProject(context.TODO(), client.GetAccessGroupProjectRequest{
			TeamID:        testTeam(),
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
  %[1]s
  name = "test-acc-%[2]s"
}

resource "vercel_access_group" "test" {
	%[1]s
	name = "test-acc-%[2]s"
}

resource "vercel_access_group_project" "test" {
	%[1]s
	project_id = vercel_project.test.id
	access_group_id = vercel_access_group.test.id
	role = "ADMIN"
}
`, teamIDConfig(), name)
}

func testAccResourceAccessGroupProjectUpdated(name string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  %[1]s
  name = "test-acc-%[2]s"
}

resource "vercel_access_group" "test" {
	%[1]s
	name = "test-acc-%[2]s"
}

resource "vercel_access_group_project" "test" {
	%[1]s
	project_id = vercel_project.test.id
	access_group_id = vercel_access_group.test.id
	role = "PROJECT_DEVELOPER"
}
`, teamIDConfig(), name)
}
