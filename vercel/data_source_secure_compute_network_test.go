package vercel_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_SecureComputeNetworkDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(`
					data "vercel_secure_compute_network" "test" {
						name = "network 1"
					}
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_secure_compute_network.test", "name", "network 1"),
					resource.TestCheckResourceAttr("data.vercel_secure_compute_network.test", "team_id", testTeam(t)),
					resource.TestCheckResourceAttrSet("data.vercel_secure_compute_network.test", "id"),
					resource.TestCheckResourceAttr("data.vercel_secure_compute_network.test", "dc", "sfo1"),
					resource.TestCheckResourceAttrSet("data.vercel_secure_compute_network.test", "projects_count"),
					resource.TestCheckResourceAttrSet("data.vercel_secure_compute_network.test", "peering_connections_count"),
					resource.TestCheckResourceAttrSet("data.vercel_secure_compute_network.test", "cidr_block"),
					resource.TestCheckResourceAttrSet("data.vercel_secure_compute_network.test", "version"),
					resource.TestCheckResourceAttr("data.vercel_secure_compute_network.test", "configuration_status", "ready"),
					resource.TestCheckResourceAttrSet("data.vercel_secure_compute_network.test", "aws.account_id"),
					resource.TestCheckResourceAttr("data.vercel_secure_compute_network.test", "aws.region", "us-west-1"),
					resource.TestCheckResourceAttr("data.vercel_secure_compute_network.test", "aws.elastic_ip_addresses.#", "2"),
					resource.TestCheckResourceAttrSet("data.vercel_secure_compute_network.test", "aws.lambda_role_arn"),
					resource.TestCheckResourceAttrSet("data.vercel_secure_compute_network.test", "aws.security_group_id"),
					resource.TestCheckResourceAttrSet("data.vercel_secure_compute_network.test", "aws.stack_id"),
					resource.TestCheckResourceAttr("data.vercel_secure_compute_network.test", "aws.subnet_ids.#", "2"),
					resource.TestCheckResourceAttrSet("data.vercel_secure_compute_network.test", "aws.vpc_id"),
				),
			},
		},
	})
}
