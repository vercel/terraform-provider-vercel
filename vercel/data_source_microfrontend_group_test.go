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
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(fmt.Sprintf(`
					resource "vercel_project" "test_project_1" {
						name = "test-acc-project-%[1]s"
					}
					resource "vercel_project" "test_project_2" {
						name = "test-acc-project-2-%[1]s"
					}
					resource "vercel_microfrontend_group" "test_group" {
						name         = "test-acc-microfrontend-group-%[1]s"
						default_app  = {
							project_id = vercel_project.test_project_1.id
						}
					}
					resource "vercel_microfrontend_group_membership" "test_child" {
						project_id             = vercel_project.test_project_2.id
						microfrontend_group_id = vercel_microfrontend_group.test_group.id
					}
					data "vercel_microfrontend_group" "test_group" {
						id = vercel_microfrontend_group.test_group.id
					}
					data "vercel_microfrontend_group_membership" "test_child" {
						microfrontend_group_id = vercel_microfrontend_group.test_group.id
						project_id = vercel_project.test_project_2.id
					}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_microfrontend_group.test_group", "name", "test-acc-microfrontend-group-"+name),
					resource.TestCheckResourceAttrSet("data.vercel_microfrontend_group.test_group", "default_app.project_id"),
					resource.TestCheckResourceAttr("data.vercel_microfrontend_group_membership.test_child", "%", "6"),
				),
			},
		},
	})
}
