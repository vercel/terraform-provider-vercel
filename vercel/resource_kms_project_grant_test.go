package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAcc_KMSProjectGrantResource(t *testing.T) {
	nameSuffix := acctest.RandString(8)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccKMSProjectGrant(nameSuffix, `["production"]`)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_kms_project_grant.test", "id"),
					resource.TestCheckResourceAttrPair("vercel_kms_project_grant.test", "issuer_id", "vercel_kms_issuer.test", "id"),
					resource.TestCheckResourceAttrPair("vercel_kms_project_grant.test", "project_id", "vercel_project.test", "id"),
					resource.TestCheckResourceAttr("vercel_kms_project_grant.test", "environments.#", "1"),
					resource.TestCheckResourceAttr("vercel_kms_project_grant.test", "environments.0", "production"),
				),
			},
			{
				Config: cfg(testAccKMSProjectGrant(nameSuffix, `["production", "preview"]`)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_kms_project_grant.test", "environments.#", "2"),
				),
			},
			{
				ResourceName:      "vercel_kms_project_grant.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getKMSProjectGrantImportID("vercel_kms_project_grant.test"),
			},
		},
	})
}

func getKMSProjectGrantImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}
		return fmt.Sprintf("%s/%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.Attributes["issuer_id"], rs.Primary.Attributes["project_id"]), nil
	}
}

func testAccKMSProjectGrant(nameSuffix, environments string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-kms-%[1]s"
}

resource "vercel_kms_issuer" "test" {
  name = "test-acc-%[1]s"
}

resource "vercel_kms_project_grant" "test" {
  issuer_id    = vercel_kms_issuer.test.id
  project_id   = vercel_project.test.id
  environments = %[2]s
}
`, nameSuffix, environments)
}
