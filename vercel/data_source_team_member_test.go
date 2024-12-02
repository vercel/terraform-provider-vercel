package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_TeamMemberDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTeamMemberDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.vercel_team_member.test", "team_id"),
					resource.TestCheckResourceAttrSet("data.vercel_team_member.test", "user_id"),
					resource.TestCheckResourceAttr("data.vercel_team_member.test", "role", "MEMBER"),
				),
			},
		},
	})
}

func testAccTeamMemberDataSourceConfig() string {
	return fmt.Sprintf(`
resource "vercel_team_member" "test" {
  %[1]s
  user_id = "%s"
  role    = "MEMBER"
}

data "vercel_team_member" "test" {
    user_id = vercel_team_member.test.user_id
    team_id = vercel_team_member.test.team_id
}
`, teamIDConfig(), testAdditionalUser())
}
