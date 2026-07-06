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

func TestAcc_AccessGroupMemberResource(t *testing.T) {
	name := acctest.RandString(16)

	// Use the authenticated user, who is a confirmed member of the testing
	// team. The access group endpoint rejects members whose invitation has not
	// been confirmed, so a freshly-invited member cannot be used here.
	user, err := testClient(t).GetUser(context.Background())
	if err != nil {
		t.Fatalf("could not get authenticated user: %s", err)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccAccessGroupMemberDoesNotExist(testClient(t), testTeam(t), "vercel_access_group_member.test"),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccResourceAccessGroupMember(name, user.ID)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAccessGroupMemberExists(testClient(t), testTeam(t), "vercel_access_group_member.test"),
					resource.TestCheckResourceAttrSet("vercel_access_group_member.test", "access_group_id"),
					resource.TestCheckResourceAttr("vercel_access_group_member.test", "user_id", user.ID),
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

func testAccResourceAccessGroupMember(name, userID string) string {
	return fmt.Sprintf(`
resource "vercel_access_group" "test" {
  name = "test-acc-%[1]s"
}

resource "vercel_access_group_member" "test" {
  access_group_id = vercel_access_group.test.id
  user_id         = "%[2]s"
}
`, name, userID)
}
