package vercel_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"testing"
	"time"

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

type testCertificateChain struct {
	privateKey                 string
	certificate                string
	certificateAuthorityBundle string
}

func testCustomCertificateChain(t *testing.T, domain string) testCertificateChain {
	t.Helper()

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating CA private key: %s", err)
	}

	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating leaf private key: %s", err)
	}

	now := time.Now()
	caSerial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("generating CA serial number: %s", err)
	}
	leafSerial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("generating leaf serial number: %s", err)
	}

	ca := x509.Certificate{
		SerialNumber: caSerial,
		Subject: pkix.Name{
			CommonName: "terraform-provider-vercel-test-ca",
		},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, &ca, &ca, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("creating CA certificate: %s", err)
	}

	leaf := x509.Certificate{
		SerialNumber: leafSerial,
		Subject: pkix.Name{
			CommonName: domain,
		},
		DNSNames:              []string{domain},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, &leaf, &ca, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("creating leaf certificate: %s", err)
	}

	return testCertificateChain{
		privateKey: string(pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(leafKey),
		})),
		certificate: string(pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: leafDER,
		})),
		certificateAuthorityBundle: string(pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: caDER,
		})),
	}
}

func TestAcc_CustomCertificateResource(t *testing.T) {
	certificate := testCustomCertificateChain(t, testDomain(t))

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
%[3]s
EOT
				}
				`, certificate.privateKey, certificate.certificate, certificate.certificateAuthorityBundle)),
				Check: resource.TestCheckResourceAttrSet("vercel_custom_certificate.test", "id"),
			},
		},
	})
}
