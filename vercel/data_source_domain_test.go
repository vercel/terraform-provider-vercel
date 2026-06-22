package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_DomainDataSource(t *testing.T) {
	domain := acctest.RandString(20) + ".terraform-ci"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccDomainDestroy(testClient(t), testTeam(t), domain),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccDomainDataSourceConfig(domain)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_domain.test", "name", domain),
					// The data source should mirror every field exposed by the resource.
					resource.TestCheckResourceAttrPair("data.vercel_domain.test", "id", "vercel_domain.test", "id"),
					resource.TestCheckResourceAttrPair("data.vercel_domain.test", "team_id", "vercel_domain.test", "team_id"),
					resource.TestCheckResourceAttrPair("data.vercel_domain.test", "zone", "vercel_domain.test", "zone"),
					resource.TestCheckResourceAttrPair("data.vercel_domain.test", "verified", "vercel_domain.test", "verified"),
					resource.TestCheckResourceAttrPair("data.vercel_domain.test", "nameservers.#", "vercel_domain.test", "nameservers.#"),
					resource.TestCheckResourceAttrPair("data.vercel_domain.test", "intended_nameservers.#", "vercel_domain.test", "intended_nameservers.#"),
				),
			},
		},
	})
}

func testAccDomainDataSourceConfig(domain string) string {
	return fmt.Sprintf(`
resource "vercel_domain" "test" {
  name = "%[1]s"
  zone = true
}

data "vercel_domain" "test" {
  name = vercel_domain.test.name
}
`, domain)
}
