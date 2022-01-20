package vercel_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAcc_DomainDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_domain.test", "suffix", "false"),
					resource.TestCheckResourceAttr("data.vercel_domain.test", "verified", "true"),
					resource.TestCheckResourceAttr("data.vercel_domain.test", "creator.email", "doug@vercel.com"),
				),
			},
		},
	})
}

func testAccDomainConfig() string {
	return `
data "vercel_domain" "test" {
    name = "dgls.dev"
}
`
}
