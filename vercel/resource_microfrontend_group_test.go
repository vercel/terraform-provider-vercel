package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testCheckMicrofrontendGroupExists(teamID, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient().GetMicrofrontendGroup(context.TODO(), rs.Primary.ID, teamID)
		return err
	}
}

func testCheckMicrofrontendGroupDeleted(n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient().GetMicrofrontendGroup(context.TODO(), rs.Primary.ID, teamID)
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !(err.Error() == "microfrontend group not found") {
			return fmt.Errorf("Unexpected error checking for deleted microfrontend group: %s", err)
		}

		return nil
	}
}

func TestAcc_MicrofrontendGroupResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckMicrofrontendGroupDeleted("vercel_microfrontend_group.test", testTeam()),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "vercel_project" "test" {
				  name = "test-acc-project-%[1]s"
				  %[2]s
				}
				resource "vercel_project" "test-2" {
				  name = "test-acc-project-2-%[1]s"
				  %[2]s
				}
				resource "vercel_microfrontend_group" "test" {
					name         = "test-acc-microfrontend-group-%[1]s"
					default_app  = vercel_project.test.id
					%[2]s
				}
				resource "vercel_microfrontend_group_membership" "test" {
					project_id             = vercel_project.test.id
					microfrontend_group_id = vercel_microfrontend_group.test.id
					%[2]s
				}
				resource "vercel_microfrontend_group_membership" "test-2" {
					project_id             = vercel_project.test-2.id
					microfrontend_group_id = vercel_microfrontend_group.test.id
					%[2]s
				}
				`, name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckMicrofrontendGroupExists(testTeam(), "vercel_microfrontend_group.test"),
					resource.TestCheckResourceAttr("vercel_microfrontend_group.test", "name", "test-acc-microfrontend-group-"+name),
					resource.TestCheckResourceAttrSet("vercel_microfrontend_group.test", "id"),
					resource.TestCheckResourceAttrSet("vercel_microfrontend_group_membership.test", "project_id"),
					resource.TestCheckResourceAttrSet("vercel_microfrontend_group_membership.test-2", "microfrontend_group_id"),
				),
			},
		},
	})
}
