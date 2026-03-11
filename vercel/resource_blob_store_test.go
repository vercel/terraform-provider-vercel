package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vercel/terraform-provider-vercel/v4/client"
)

func TestAcc_BlobStoreResource(t *testing.T) {
	suffix := acctest.RandString(16)
	initialName := fmt.Sprintf("test-acc-blob-%s", suffix)
	updatedName := fmt.Sprintf("test-acc-blob-updated-%s", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckBlobStoreDeleted(testClient(t), "vercel_blob_store.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccBlobStoreResourceConfig(initialName, "public")),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckBlobStoreExists(testClient(t), testTeam(t), "vercel_blob_store.test"),
					resource.TestCheckResourceAttr("vercel_blob_store.test", "name", initialName),
					resource.TestCheckResourceAttr("vercel_blob_store.test", "access", "public"),
					resource.TestCheckResourceAttr("vercel_blob_store.test", "region", "iad1"),
					resource.TestCheckResourceAttr("vercel_blob_store.test", "team_id", testTeam(t)),
					resource.TestCheckResourceAttrSet("vercel_blob_store.test", "id"),
					resource.TestCheckResourceAttrSet("vercel_blob_store.test", "status"),
					resource.TestCheckResourceAttrSet("vercel_blob_store.test", "size"),
					resource.TestCheckResourceAttrSet("vercel_blob_store.test", "file_count"),
					resource.TestCheckResourceAttrSet("vercel_blob_store.test", "created_at"),
					resource.TestCheckResourceAttrSet("vercel_blob_store.test", "updated_at"),
				),
			},
			{
				ResourceName:      "vercel_blob_store.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getBlobStoreImportID("vercel_blob_store.test"),
			},
			{
				Config: cfg(testAccBlobStoreResourceConfig(updatedName, "public")),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("vercel_blob_store.test", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckBlobStoreExists(testClient(t), testTeam(t), "vercel_blob_store.test"),
					resource.TestCheckResourceAttr("vercel_blob_store.test", "name", updatedName),
					resource.TestCheckResourceAttr("vercel_blob_store.test", "access", "public"),
					resource.TestCheckResourceAttr("vercel_blob_store.test", "region", "iad1"),
				),
			},
			{
				Config: cfg(testAccBlobStoreResourceConfig(updatedName, "private")),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("vercel_blob_store.test", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckBlobStoreExists(testClient(t), testTeam(t), "vercel_blob_store.test"),
					resource.TestCheckResourceAttr("vercel_blob_store.test", "name", updatedName),
					resource.TestCheckResourceAttr("vercel_blob_store.test", "access", "private"),
					resource.TestCheckResourceAttr("vercel_blob_store.test", "region", "iad1"),
				),
			},
		},
	})
}

func testAccBlobStoreResourceConfig(name, access string) string {
	return fmt.Sprintf(`
resource "vercel_blob_store" "test" {
  name   = "%s"
  access = "%s"
}
`, name, access)
}

func getBlobStoreImportID(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return "", fmt.Errorf("no ID is set")
		}

		teamID := rs.Primary.Attributes["team_id"]
		if teamID == "" {
			return rs.Primary.ID, nil
		}

		return fmt.Sprintf("%s/%s", teamID, rs.Primary.ID), nil
	}
}

func testCheckBlobStoreExists(testClient *client.Client, teamID, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetBlobStore(context.TODO(), rs.Primary.ID, teamID)
		return err
	}
}

func testCheckBlobStoreDeleted(testClient *client.Client, resourceName, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetBlobStore(context.TODO(), rs.Primary.ID, teamID)
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("unexpected error checking for deleted blob store: %s", err)
		}

		return nil
	}
}
