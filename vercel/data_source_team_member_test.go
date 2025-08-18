package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_TeamMemberDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccTeamMemberDataSourceConfig(testAdditionalUserEmail(t), testTeam(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.vercel_team_member.test", "team_id"),
					resource.TestCheckResourceAttrSet("data.vercel_team_member.test", "user_id"),
					resource.TestCheckResourceAttr("data.vercel_team_member.test", "role", "MEMBER"),
				),
			},
		},
	})
}

func testAccTeamMemberDataSourceConfig(userEmail, teamID string) string {
	return fmt.Sprintf(`
resource "vercel_team_member" "test" {
  email   = "%[1]s"
  team_id = "%[2]s"
  role    = "MEMBER"
}

data "vercel_team_member" "test" {
    user_id = vercel_team_member.test.user_id
    team_id = vercel_team_member.test.team_id
}
`, userEmail, teamID)
}
