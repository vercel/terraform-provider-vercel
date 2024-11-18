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

func TestAcc_AccessGroupResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testCheckAccessGroupDoesNotExist("vercel_access_group.test"),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceAccessGroup(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAccessGroupExists("vercel_access_group.test"),
					resource.TestCheckResourceAttrSet("vercel_access_group.test", "id"),
					resource.TestCheckResourceAttr("vercel_access_group.test", "name", fmt.Sprintf("test-acc-%s", name)),
				),
			},
			{
				ResourceName:      "vercel_access_group.test",
				ImportState:       true,
				ImportStateIdFunc: getAccessGroupImportID("vercel_access_group.test"),
			},
			{
				Config: testAccResourceAccessGroupUpdated(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAccessGroupExists("vercel_access_group.test"),
					resource.TestCheckResourceAttrSet("vercel_access_group.test", "id"),
					resource.TestCheckResourceAttr("vercel_access_group.test", "name", fmt.Sprintf("test-acc-%s-updated", name)),
				),
			},
		},
	})
}

func getAccessGroupImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return "", fmt.Errorf("no ID is set")
		}

		return fmt.Sprintf(
			"%s/%s",
			rs.Primary.Attributes["team_id"],
			rs.Primary.Attributes["access_group_id"],
		), nil
	}
}

func testCheckAccessGroupExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient().GetAccessGroup(context.TODO(), client.GetAccessGroupRequest{
			TeamID:        testTeam(),
			AccessGroupID: rs.Primary.ID,
		})
		return err
	}
}

func testCheckAccessGroupDoesNotExist(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient().GetAccessGroup(context.TODO(), client.GetAccessGroupRequest{
			TeamID:        testTeam(),
			AccessGroupID: rs.Primary.ID,
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

func testAccResourceAccessGroup(name string) string {
	return fmt.Sprintf(`
resource "vercel_access_group" "test" {
	%[1]s
  name = "test-acc-%[2]s"
}
`, teamIDConfig(), name)
}

func testAccResourceAccessGroupUpdated(name string) string {
	return fmt.Sprintf(`
resource "vercel_access_group" "test" {
  %[1]s
  name  = "test-acc-%[2]s-updated"
}
`, teamIDConfig(), name)
}
