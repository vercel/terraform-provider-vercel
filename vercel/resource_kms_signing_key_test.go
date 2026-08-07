package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAcc_KMSSigningKeyResource(t *testing.T) {
	testAccKMSPreCheck(t)
	nameSuffix := acctest.RandString(8)

	var firstKeyID string
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccKMSSigningKey(nameSuffix, "1")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_kms_signing_key.test", "id"),
					resource.TestCheckResourceAttrPair("vercel_kms_signing_key.test", "issuer_id", "vercel_kms_issuer.test", "id"),
					resource.TestCheckResourceAttrSet("vercel_kms_signing_key.test", "public_key"),
					resource.TestCheckResourceAttr("vercel_kms_signing_key.test", "status", "active"),
					captureKMSSigningKeyID("vercel_kms_signing_key.test", &firstKeyID),
				),
			},
			{
				// Changing keepers forces a new rotation, producing a new key id.
				Config: cfg(testAccKMSSigningKey(nameSuffix, "2")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_kms_signing_key.test", "id"),
					checkKMSSigningKeyChanged("vercel_kms_signing_key.test", &firstKeyID),
				),
			},
		},
	})
}

func captureKMSSigningKeyID(n string, target *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		*target = rs.Primary.ID
		return nil
	}
}

func checkKMSSigningKeyChanged(n string, previous *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		if rs.Primary.ID == *previous {
			return fmt.Errorf("expected a new signing key id after changing keepers, but it stayed %s", *previous)
		}
		return nil
	}
}

func testAccKMSSigningKey(nameSuffix, rotation string) string {
	return fmt.Sprintf(`
resource "vercel_kms_issuer" "test" {
  name = "test-acc-%[1]s"
}

resource "vercel_kms_signing_key" "test" {
  issuer_id = vercel_kms_issuer.test.id
  keepers = {
    rotation = "%[2]s"
  }
}
`, nameSuffix, rotation)
}
