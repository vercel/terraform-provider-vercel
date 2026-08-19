package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAcc_KMSSigningKeyResource(t *testing.T) {
	nameSuffix := acctest.RandString(8)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Applying the resource rotates a brand-new signing key into the
				// issuer. We cannot assert a second, keepers-triggered rotation in
				// the same test: a rotated key stays `pending` through a multi-minute
				// activation window, and the API allows only one pending key per
				// issuer at a time, so a back-to-back rotation is rejected with
				// `signing_key_activation_pending`.
				Config: cfg(testAccKMSSigningKey(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_kms_signing_key.test", "id"),
					resource.TestCheckResourceAttrPair("vercel_kms_signing_key.test", "issuer_id", "vercel_kms_issuer.test", "id"),
					resource.TestCheckResourceAttrSet("vercel_kms_signing_key.test", "public_key_jwk"),
					// A freshly rotated key is `pending` until its public key
					// propagates and it is activated, so only assert it is set.
					resource.TestCheckResourceAttrSet("vercel_kms_signing_key.test", "status"),
					// The rotated key must be distinct from the issuer's initial key.
					checkKMSSigningKeyRotated("vercel_kms_signing_key.test", "vercel_kms_issuer.test"),
				),
			},
		},
	})
}

// checkKMSSigningKeyRotated asserts that the signing key resource holds a key id
// distinct from the issuer's initial key, proving the apply performed a real
// rotation. The issuer resource's state still lists only its initial key at
// check time (it is not refreshed between apply and Check), so index 0 is
// unambiguously the initial key.
func checkKMSSigningKeyRotated(keyResource, issuerResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		key, ok := s.RootModule().Resources[keyResource]
		if !ok {
			return fmt.Errorf("not found: %s", keyResource)
		}
		issuer, ok := s.RootModule().Resources[issuerResource]
		if !ok {
			return fmt.Errorf("not found: %s", issuerResource)
		}
		initial := issuer.Primary.Attributes["signing_keys.0.key_id"]
		if initial == "" {
			return fmt.Errorf("issuer %s has no initial signing key in state", issuerResource)
		}
		if key.Primary.ID == initial {
			return fmt.Errorf("expected rotated signing key id to differ from the issuer's initial key %s", initial)
		}
		return nil
	}
}

func testAccKMSSigningKey(nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_kms_issuer" "test" {
  name = "test-acc-%[1]s"
}

resource "vercel_kms_signing_key" "test" {
  issuer_id = vercel_kms_issuer.test.id
  keepers = {
    rotation = "1"
  }
}
`, nameSuffix)
}
