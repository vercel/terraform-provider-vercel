package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_MicrofrontendGroupDataSource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
						default_app  = {
							project_id = vercel_project.test.id
						}
						%[2]s
					}
					resource "vercel_microfrontend_group_membership" "test-2" {
						project_id             = vercel_project.test-2.id
						microfrontend_group_id = vercel_microfrontend_group.test.id
						%[2]s
					}
					data "vercel_microfrontend_group" "test" {
						id = vercel_microfrontend_group.test.id
						%[2]s
					}
					data "vercel_microfrontend_group_membership" "test-2" {
						microfrontend_group_id = vercel_microfrontend_group.test.id
						project_id = vercel_microfrontend_group_membership.test-2.project_id
						%[2]s
					}	
				`, name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_microfrontend_group.test", "name", "long-term-test"),
					resource.TestCheckResourceAttr("data.vercel_microfrontend_group.test", "id", "mfe_z5wEafgq19cbB92CAQV7fZgUXAdp"),
				),
			},
		},
	})
}
