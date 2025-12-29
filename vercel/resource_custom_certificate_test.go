package vercel_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

func testCheckCustomCertificateDoesNotExist(testClient *client.Client, teamID string, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetCustomCertificate(context.TODO(), client.GetCustomCertificateRequest{
			TeamID: teamID,
			ID:     rs.Primary.ID,
		})
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("Unexpected error checking for deleted certificate: %s", err)
		}

		return nil
	}
}

func testCert(t *testing.T) string {
	v := os.Getenv("VERCEL_TERRAFORM_TESTING_CERTIFICATE")
	if v == "" {
		t.Fatalf("Missing required environment variable VERCEL_TERRAFORM_TESTING_CERTIFICATE")
	}
	return v
}

func testCertKey(t *testing.T) string {
	v := os.Getenv("VERCEL_TERRAFORM_TESTING_CERTIFICATE_KEY")
	if v == "" {
		t.Fatalf("Missing required environment variable VERCEL_TERRAFORM_TESTING_CERTIFICATE_KEY")
	}
	return v
}

func TestAcc_CustomCertificateResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckCustomCertificateDoesNotExist(testClient(t), testTeam(t), "vercel_custom_certificate.test"),
		Steps: []resource.TestStep{
			{
				Config: cfg(fmt.Sprintf(`
resource "vercel_custom_certificate" "test" {
	private_key = <<EOT
%[1]s
EOT
	certificate = <<EOT
%[2]s
EOT
	certificate_authority_certificate = <<EOT
%[2]s
EOT
				}
				`, testCertKey(t), testCert(t))),
				Check: resource.TestCheckResourceAttrSet("vercel_custom_certificate.test", "id"),
			},
		},
	})
}
