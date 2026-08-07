package vercel_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

// testAccKMSPreCheck skips KMS acceptance tests unless the testing team has the
// KMS feature enabled. KMS endpoints intentionally return 404 when the feature
// flag is off, so these tests require an opted-in team.
func testAccKMSPreCheck(t *testing.T) {
	if os.Getenv("VERCEL_TERRAFORM_TESTING_KMS") != "true" {
		t.Skip("Skipping KMS acceptance test: set VERCEL_TERRAFORM_TESTING_KMS=true on a KMS-enabled team to run")
	}
}

func testCheckKMSIssuerExists(testClient *client.Client, teamID, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetKMSIssuer(context.TODO(), rs.Primary.ID, teamID)
		if client.NotFound(err) {
			return fmt.Errorf("test failed because the KMS issuer %s %s could not be found", teamID, rs.Primary.ID)
		}
		return err
	}
}

func testAccKMSIssuerDestroy(testClient *client.Client, n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return nil
		}

		_, err := testClient.GetKMSIssuer(context.TODO(), rs.Primary.ID, teamID)
		if client.NotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("expected KMS issuer %s to be destroyed but it still exists", rs.Primary.ID)
	}
}

func TestAcc_KMSIssuerResource(t *testing.T) {
	testAccKMSPreCheck(t)
	nameSuffix := acctest.RandString(8)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccKMSIssuerDestroy(testClient(t), "vercel_kms_issuer.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccKMSIssuer(nameSuffix, "RS256")),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckKMSIssuerExists(testClient(t), testTeam(t), "vercel_kms_issuer.test"),
					resource.TestCheckResourceAttrSet("vercel_kms_issuer.test", "id"),
					resource.TestCheckResourceAttr("vercel_kms_issuer.test", "name", fmt.Sprintf("test-acc-%s", nameSuffix)),
					resource.TestCheckResourceAttr("vercel_kms_issuer.test", "algorithm", "RS256"),
					resource.TestCheckResourceAttr("vercel_kms_issuer.test", "origin", "vercel"),
					resource.TestCheckResourceAttrSet("vercel_kms_issuer.test", "owner_id"),
					resource.TestCheckResourceAttrSet("vercel_kms_issuer.test", "signing_keys.0.key_id"),
					resource.TestCheckResourceAttrSet("vercel_kms_issuer.test", "signing_keys.0.public_key"),
				),
			},
			{
				ResourceName:      "vercel_kms_issuer.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getKMSIssuerImportID("vercel_kms_issuer.test"),
			},
			{
				Config: cfg(testAccKMSIssuer(nameSuffix+"-renamed", "RS256")),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckKMSIssuerExists(testClient(t), testTeam(t), "vercel_kms_issuer.test"),
					resource.TestCheckResourceAttr("vercel_kms_issuer.test", "name", fmt.Sprintf("test-acc-%s-renamed", nameSuffix)),
				),
			},
		},
	})
}

func TestAcc_KMSIssuerDataSource(t *testing.T) {
	testAccKMSPreCheck(t)
	nameSuffix := acctest.RandString(8)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccKMSIssuerDataSource(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.vercel_kms_issuer.test", "id", "vercel_kms_issuer.test", "id"),
					resource.TestCheckResourceAttr("data.vercel_kms_issuer.test", "name", fmt.Sprintf("test-acc-%s", nameSuffix)),
					resource.TestCheckResourceAttrSet("data.vercel_kms_issuer.test", "algorithm"),
					resource.TestCheckResourceAttrSet("data.vercel_kms_issuer.test", "signing_keys.0.key_id"),
				),
			},
		},
	})
}

func getKMSIssuerImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return "", fmt.Errorf("no ID is set")
		}
		return fmt.Sprintf("%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.ID), nil
	}
}

func testAccKMSIssuer(nameSuffix, algorithm string) string {
	return fmt.Sprintf(`
resource "vercel_kms_issuer" "test" {
  name      = "test-acc-%[1]s"
  algorithm = "%[2]s"
}
`, nameSuffix, algorithm)
}

func testAccKMSIssuerDataSource(nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_kms_issuer" "test" {
  name = "test-acc-%[1]s"
}

data "vercel_kms_issuer" "test" {
  id = vercel_kms_issuer.test.id
}
`, nameSuffix)
}
