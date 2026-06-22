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
					resource.TestCheckResourceAttrSet("data.vercel_domain.test", "id"),
					resource.TestCheckResourceAttrSet("data.vercel_domain.test", "team_id"),
				),
			},
		},
	})
}

func testAccDomainDataSourceConfig(domain string) string {
	return fmt.Sprintf(`
resource "vercel_domain" "test" {
  name = "%[1]s"
}

data "vercel_domain" "test" {
  name = vercel_domain.test.name
}
`, domain)
}
