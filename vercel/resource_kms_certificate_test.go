package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAcc_KMSCertificateResource(t *testing.T) {
	testAccKMSPreCheck(t)
	nameSuffix := acctest.RandString(8)

	var firstSerial string
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccKMSCertificate(nameSuffix, "1")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_kms_certificate.test", "certificate"),
					resource.TestCheckResourceAttrSet("vercel_kms_certificate.test", "serial_number"),
					resource.TestCheckResourceAttrSet("vercel_kms_certificate.test", "key_id"),
					resource.TestCheckResourceAttrSet("vercel_kms_certificate.test", "kms_issuer_url"),
					resource.TestCheckResourceAttrPair("vercel_kms_certificate.test", "id", "vercel_kms_certificate.test", "serial_number"),
					captureKMSCertificateSerial("vercel_kms_certificate.test", &firstSerial),
				),
			},
			{
				// Changing keepers mints a fresh certificate with a new serial.
				Config: cfg(testAccKMSCertificate(nameSuffix, "2")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("vercel_kms_certificate.test", "serial_number"),
					checkKMSCertificateSerialChanged("vercel_kms_certificate.test", &firstSerial),
				),
			},
		},
	})
}

func captureKMSCertificateSerial(n string, target *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		*target = rs.Primary.Attributes["serial_number"]
		return nil
	}
}

func checkKMSCertificateSerialChanged(n string, previous *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		if rs.Primary.Attributes["serial_number"] == *previous {
			return fmt.Errorf("expected a new certificate serial after changing keepers, but it stayed %s", *previous)
		}
		return nil
	}
}

func testAccKMSCertificate(nameSuffix, rotation string) string {
	return fmt.Sprintf(`
resource "vercel_kms_issuer" "test" {
  name = "test-acc-%[1]s"
}

resource "vercel_kms_certificate" "test" {
  issuer_id = vercel_kms_issuer.test.id
  keepers = {
    rotation = "%[2]s"
  }

  subject = {
    ou = "Engineering"
    c  = "US"
  }
}
`, nameSuffix, rotation)
}
