package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_DomainConfigDataSource(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	domain := acctest.RandString(30) + ".example.com"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             noopDestroyCheck,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccDomainConfigDataSourceConfig(projectSuffix, domain)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_domain_config.test", "domain", domain),
					resource.TestCheckResourceAttrSet("data.vercel_domain_config.test", "project_id_or_name"),
					resource.TestCheckResourceAttrSet("data.vercel_domain_config.test", "recommended_cname"),
					resource.TestCheckResourceAttrSet("data.vercel_domain_config.test", "recommended_ipv4s.#"),
				),
			},
		},
	})
}

func testAccDomainConfigDataSourceConfig(projectSuffix, domain string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-domain-config-%s"
}

data "vercel_domain_config" "test" {
  domain = "%s"
  project_id_or_name = vercel_project.test.id
}
`, projectSuffix, domain)
}
