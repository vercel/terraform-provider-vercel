package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func getTeamMemberImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}

		return fmt.Sprintf("%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.Attributes["user_id"]), nil
	}
}

func TestAcc_TeamMemberResource(t *testing.T) {
	randomSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: cfg(testAccTeamMemberResourceConfig("MEMBER", testAdditionalUser(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_team_member.test", "team_id"),
					resource.TestCheckResourceAttrSet("vercel_team_member.test", "user_id"),
					resource.TestCheckResourceAttr("vercel_team_member.test", "role", "MEMBER"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "vercel_team_member.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateIdFunc:                    getTeamMemberImportID("vercel_team_member.test"),
				ImportStateVerifyIdentifierAttribute: "user_id",
			},
			// Update testing
			{
				Config: cfg(testAccTeamMemberResourceConfig("VIEWER", testAdditionalUser(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_team_member.test", "team_id"),
					resource.TestCheckResourceAttrSet("vercel_team_member.test", "user_id"),
					resource.TestCheckResourceAttr("vercel_team_member.test", "role", "VIEWER"),
				),
			},
			// Test with projects
			{
				Config: cfg(testAccTeamMemberResourceConfigWithProjects(randomSuffix, testAdditionalUser(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_team_member.test", "team_id"),
					resource.TestCheckResourceAttrSet("vercel_team_member.test", "user_id"),
					resource.TestCheckResourceAttr("vercel_team_member.test", "role", "CONTRIBUTOR"),
					resource.TestCheckResourceAttr("vercel_team_member.test", "projects.#", "1"),
				),
			},
			// Test with access groups
			{
				Config: cfg(testAccTeamMemberResourceConfigWithAccessGroups(randomSuffix, testAdditionalUser(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_team_member.test", "team_id"),
					resource.TestCheckResourceAttrSet("vercel_team_member.test", "user_id"),
					resource.TestCheckResourceAttr("vercel_team_member.test", "role", "CONTRIBUTOR"),
					resource.TestCheckResourceAttr("vercel_team_member.test", "access_groups.#", "1"),
				),
			},
		},
	})
}

func testAccTeamMemberResourceConfig(role string, user string) string {
	return fmt.Sprintf(`
resource "vercel_team_member" "test" {
  user_id = "%s"
  role    = "%s"
}
`, user, role)
}

func testAccTeamMemberResourceConfigWithProjects(randomSuffix string, user string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-example-project-%[1]s"
}

resource "vercel_team_member" "test" {
  user_id = "%[2]s"
  role    = "CONTRIBUTOR"
  projects = [{
    project_id = vercel_project.test.id
    role       = "PROJECT_VIEWER"
  }]
}
`, randomSuffix, user)
}

func testAccTeamMemberResourceConfigWithAccessGroups(randomSuffix string, user string) string {
	return fmt.Sprintf(`
resource "vercel_access_group" "test" {
    name = "test-acc-access-group-%[2]s"
}

resource "vercel_team_member" "test" {
  user_id = "%[1]s"
  role    = "CONTRIBUTOR"

  access_groups = [vercel_access_group.test.id]
}
`, user, randomSuffix)
}
