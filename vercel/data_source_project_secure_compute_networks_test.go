package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_ProjectSecureComputeNetworksDataSource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(fmt.Sprintf(`
				resource "vercel_project" "test" {
				  name = "test-acc-project-%[1]s"
				}
				data "vercel_secure_compute_network" "test" {
					name = "network 1"
				}
				data "vercel_secure_compute_network" "test_2" {
					name = "network 2"
				}
				resource "vercel_project_secure_compute_networks" "test" {
					project_id = vercel_project.test.id
					secure_compute_networks = [
						{
							environment    = "production"
							network_id     = data.vercel_secure_compute_network.test.id
							passive        = true
							builds_enabled = false
						},
						{
							environment    = "preview"
							network_id     = data.vercel_secure_compute_network.test_2.id
							passive        = false
							builds_enabled = false
						}
					]
				}

				data "vercel_project_secure_compute_networks" "test" {
					project_id = vercel_project_secure_compute_networks.test.project_id
				}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.vercel_project_secure_compute_networks.test", "project_id"),
					resource.TestCheckResourceAttrSet("data.vercel_project_secure_compute_networks.test", "team_id"),
					resource.TestCheckResourceAttr("data.vercel_project_secure_compute_networks.test", "secure_compute_networks.#", "2"),
				),
			},
		},
	})
}
