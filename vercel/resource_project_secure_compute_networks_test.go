package vercel_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

func testCheckProjectSecureComputeNetworksDeleted(testClient *client.Client, n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no project ID is set")
		}

		project, err := testClient.GetProject(context.TODO(), rs.Primary.ID, teamID)
		if err != nil {
			return fmt.Errorf("unexpected error %w", err)
		}

		if len(project.ConnectConfigurations) > 0 {
			return fmt.Errorf("expected no connect configurations, but got %d", len(project.ConnectConfigurations))
		}

		return nil
	}
}

func TestAcc_ProjectSecureComputeNetworksResource(t *testing.T) {
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
				resource "vercel_project_secure_compute_networks" "test" {
					project_id = vercel_project.test.id
					secure_compute_networks = [
						{
							environment    = "production"
							network_id     = data.vercel_secure_compute_network.test.id
							passive        = true
							builds_enabled = true
						}
					]
				}
				`, name)),
				ExpectError: regexp.MustCompile("builds_enabled cannot be `true` if passive is `true`"),
			},
			{
				Config: cfg(fmt.Sprintf(`
				resource "vercel_project" "test" {
				  name = "test-acc-project-%[1]s"
				}
				data "vercel_secure_compute_network" "test" {
					name = "network 1"
				}
				resource "vercel_project_secure_compute_networks" "test" {
					project_id = vercel_project.test.id
					secure_compute_networks = [
						{
							environment    = "production"
							network_id     = data.vercel_secure_compute_network.test.id
							passive        = false
							builds_enabled = true
						}
					]
				}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_project_secure_compute_networks.test", "project_id"),
					resource.TestCheckResourceAttr("vercel_project_secure_compute_networks.test", "secure_compute_networks.#", "1"),
					resource.TestCheckResourceAttr("vercel_project_secure_compute_networks.test", "secure_compute_networks.0.environment", "production"),
					resource.TestCheckResourceAttrSet("vercel_project_secure_compute_networks.test", "secure_compute_networks.0.network_id"),
					resource.TestCheckResourceAttr("vercel_project_secure_compute_networks.test", "secure_compute_networks.0.passive", "false"),
					resource.TestCheckResourceAttr("vercel_project_secure_compute_networks.test", "secure_compute_networks.0.builds_enabled", "true"),
				),
			},
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
							environment    = "preview"
							network_id     = data.vercel_secure_compute_network.test.id
							passive        = true
							builds_enabled = false
						},
						{
							environment    = "production"
							network_id     = data.vercel_secure_compute_network.test_2.id
							passive        = true
							builds_enabled = false
						},
					]
				}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_project_secure_compute_networks.test", "project_id"),
					resource.TestCheckResourceAttr("vercel_project_secure_compute_networks.test", "secure_compute_networks.#", "2"),
					resource.TestCheckResourceAttrSet("vercel_project_secure_compute_networks.test", "secure_compute_networks.0.environment"),
					resource.TestCheckResourceAttrSet("vercel_project_secure_compute_networks.test", "secure_compute_networks.0.network_id"),
					resource.TestCheckResourceAttr("vercel_project_secure_compute_networks.test", "secure_compute_networks.0.passive", "true"),
					resource.TestCheckResourceAttr("vercel_project_secure_compute_networks.test", "secure_compute_networks.0.builds_enabled", "false"),
					resource.TestCheckResourceAttrSet("vercel_project_secure_compute_networks.test", "secure_compute_networks.1.environment"),
					resource.TestCheckResourceAttrSet("vercel_project_secure_compute_networks.test", "secure_compute_networks.1.network_id"),
					resource.TestCheckResourceAttr("vercel_project_secure_compute_networks.test", "secure_compute_networks.1.passive", "true"),
					resource.TestCheckResourceAttr("vercel_project_secure_compute_networks.test", "secure_compute_networks.1.builds_enabled", "false"),
				),
			},
			{
				Config: cfg(fmt.Sprintf(`
				resource "vercel_project" "test" {
				  name = "test-acc-project-%[1]s"
				}
				`, name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckProjectSecureComputeNetworksDeleted(testClient(t), "vercel_project.test", testTeam(t)),
				),
			},
		},
	})
}
