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

func TestAcc_AccessGroupMemberResource(t *testing.T) {
	name := acctest.RandString(16)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccAccessGroupMemberDoesNotExist(testClient(t), testTeam(t), "vercel_access_group_member.test"),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccResourceAccessGroupMember(name, testAdditionalUserEmail(t), testTeam(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAccessGroupMemberExists(testClient(t), testTeam(t), "vercel_access_group_member.test"),
					resource.TestCheckResourceAttrSet("vercel_access_group_member.test", "access_group_id"),
					resource.TestCheckResourceAttrSet("vercel_access_group_member.test", "user_id"),
				),
			},
			{
				ResourceName:      "vercel_access_group_member.test",
				ImportState:       true,
				ImportStateIdFunc: getAccessGroupMemberImportID("vercel_access_group_member.test"),
			},
		},
	})
}

func getAccessGroupMemberImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}

		return fmt.Sprintf(
			"%s/%s/%s",
			rs.Primary.Attributes["team_id"],
			rs.Primary.Attributes["access_group_id"],
			rs.Primary.Attributes["user_id"],
		), nil
	}
}

func testCheckAccessGroupMemberExists(testClient *client.Client, teamID string, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		_, err := testClient.GetAccessGroupMember(context.TODO(), client.GetAccessGroupMemberRequest{
			TeamID:        teamID,
			AccessGroupID: rs.Primary.Attributes["access_group_id"],
			UserID:        rs.Primary.Attributes["user_id"],
		})
		return err
	}
}

func testAccAccessGroupMemberDoesNotExist(testClient *client.Client, teamID string, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		_, err := testClient.GetAccessGroupMember(context.TODO(), client.GetAccessGroupMemberRequest{
			TeamID:        teamID,
			AccessGroupID: rs.Primary.Attributes["access_group_id"],
			UserID:        rs.Primary.Attributes["user_id"],
		})
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("Unexpected error checking for deleted access group member: %s", err)
		}

		return nil
	}
}

func testAccResourceAccessGroupMember(name, userEmail, teamID string) string {
	return fmt.Sprintf(`
resource "vercel_team_member" "test" {
  email   = "%[2]s"
  team_id = "%[3]s"
  role    = "MEMBER"
}

resource "vercel_access_group" "test" {
  name = "test-acc-%[1]s"
}

resource "vercel_access_group_member" "test" {
  access_group_id = vercel_access_group.test.id
  user_id         = vercel_team_member.test.user_id
}
`, name, userEmail, teamID)
}
