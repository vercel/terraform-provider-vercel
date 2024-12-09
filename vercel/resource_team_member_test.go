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
	t.Parallel()
	randomSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTeamMemberResourceConfig("MEMBER"),
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
				Config: testAccTeamMemberResourceConfig("VIEWER"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_team_member.test", "team_id"),
					resource.TestCheckResourceAttrSet("vercel_team_member.test", "user_id"),
					resource.TestCheckResourceAttr("vercel_team_member.test", "role", "VIEWER"),
				),
			},
			// Test with projects
			{
				Config: testAccTeamMemberResourceConfigWithProjects(randomSuffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_team_member.test_with_projects", "team_id"),
					resource.TestCheckResourceAttrSet("vercel_team_member.test_with_projects", "user_id"),
					resource.TestCheckResourceAttr("vercel_team_member.test_with_projects", "role", "CONTRIBUTOR"),
					resource.TestCheckResourceAttr("vercel_team_member.test_with_projects", "projects.#", "1"),
				),
			},
			// Test with access groups
			{
				Config: testAccTeamMemberResourceConfigWithAccessGroups(randomSuffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_team_member.test_with_access_groups", "team_id"),
					resource.TestCheckResourceAttrSet("vercel_team_member.test_with_access_groups", "user_id"),
					resource.TestCheckResourceAttr("vercel_team_member.test_with_access_groups", "role", "CONTRIBUTOR"),
					resource.TestCheckResourceAttr("vercel_team_member.test_with_access_groups", "access_groups.#", "1"),
				),
			},
		},
	})
}

func testAccTeamMemberResourceConfig(role string) string {
	return fmt.Sprintf(`
resource "vercel_team_member" "test" {
  %[1]s
  user_id = "%s"
  role    = "%s"
}
`, teamIDConfig(), testAdditionalUser(), role)
}

func testAccTeamMemberResourceConfigWithProjects(randomSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-example-project-%[1]s"
  %[2]s
}

resource "vercel_team_member" "test_with_projects" {
  %[2]s
  user_id = "%s"
  role    = "CONTRIBUTOR"
  projects = [{
    project_id = vercel_project.test.id
    role       = "PROJECT_VIEWER"
  }]
}
`, randomSuffix, teamIDConfig(), testAdditionalUser())
}

func testAccTeamMemberResourceConfigWithAccessGroups(randomSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_access_group" "test" {
    %[1]s
    name = "test-acc-access-group-%[3]s"
}

resource "vercel_team_member" "test_with_access_groups" {
  %[1]s
  user_id = "%[2]s"
  role    = "CONTRIBUTOR"

  access_groups = [vercel_access_group.test.id]
}
`, teamIDConfig(), testAdditionalUser(), randomSuffix)
}