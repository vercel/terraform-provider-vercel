package vercel_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vercel/terraform-provider-vercel/v4/client"
)

func TestAcc_BlobObjectResource(t *testing.T) {
	suffix := acctest.RandString(16)
	storeName := fmt.Sprintf("test-acc-blob-obj-%s", suffix)
	sourceOne := testBlobObjectSourcePath(t, "object-one.txt")
	sourceTwo := testBlobObjectSourcePath(t, "object-two.txt")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckBlobObjectDeleted(testClient(t), "vercel_blob_object.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccBlobObjectResourceConfig(storeName, sourceOne)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckBlobObjectExists(testClient(t), testTeam(t), "vercel_blob_object.test"),
					resource.TestCheckResourceAttrPair("vercel_blob_object.test", "store_id", "vercel_blob_store.test", "id"),
					resource.TestCheckResourceAttr("vercel_blob_object.test", "team_id", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_blob_object.test", "pathname", "terraform/object.txt"),
					resource.TestCheckResourceAttr("vercel_blob_object.test", "source", sourceOne),
					resource.TestCheckResourceAttr("vercel_blob_object.test", "content_type", "text/plain"),
					resource.TestCheckResourceAttr("vercel_blob_object.test", "cache_control_max_age", "3600"),
					resource.TestCheckResourceAttr("vercel_blob_object.test", "cache_control", "public, max-age=3600"),
					resource.TestCheckResourceAttrSet("vercel_blob_object.test", "id"),
					resource.TestCheckResourceAttrSet("vercel_blob_object.test", "source_sha256"),
					resource.TestCheckResourceAttrSet("vercel_blob_object.test", "url"),
					resource.TestCheckResourceAttrSet("vercel_blob_object.test", "download_url"),
					resource.TestCheckResourceAttrSet("vercel_blob_object.test", "size"),
					resource.TestCheckResourceAttrSet("vercel_blob_object.test", "uploaded_at"),
					resource.TestCheckResourceAttrSet("vercel_blob_object.test", "content_disposition"),
					resource.TestCheckResourceAttrSet("vercel_blob_object.test", "etag"),
				),
			},
			{
				Config: cfg(testAccBlobObjectResourceUpdatedConfig(storeName, sourceTwo)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("vercel_blob_object.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckBlobObjectExists(testClient(t), testTeam(t), "vercel_blob_object.test"),
					resource.TestCheckResourceAttr("vercel_blob_object.test", "pathname", "terraform/object.txt"),
					resource.TestCheckResourceAttr("vercel_blob_object.test", "source", sourceTwo),
					resource.TestCheckResourceAttr("vercel_blob_object.test", "content_type", "text/plain"),
					resource.TestCheckResourceAttr("vercel_blob_object.test", "cache_control_max_age", "7200"),
					resource.TestCheckResourceAttr("vercel_blob_object.test", "cache_control", "public, max-age=7200"),
					resource.TestCheckResourceAttrSet("vercel_blob_object.test", "source_sha256"),
					resource.TestCheckResourceAttrSet("vercel_blob_object.test", "etag"),
				),
			},
		},
	})
}

func testAccBlobObjectResourceConfig(storeName, source string) string {
	return fmt.Sprintf(`
resource "vercel_blob_store" "test" {
  name = "%s"
}

resource "vercel_blob_object" "test" {
  store_id              = vercel_blob_store.test.id
  pathname              = "terraform/object.txt"
  source                = "%s"
  content_type          = "text/plain"
  cache_control_max_age = 3600
}
`, storeName, source)
}

func testAccBlobObjectResourceUpdatedConfig(storeName, source string) string {
	return fmt.Sprintf(`
resource "vercel_blob_store" "test" {
  name = "%s"
}

resource "vercel_blob_object" "test" {
  store_id              = vercel_blob_store.test.id
  pathname              = "terraform/object.txt"
  source                = "%s"
  content_type          = "text/plain"
  cache_control_max_age = 7200
}
`, storeName, source)
}

func testCheckBlobObjectExists(testClient *client.Client, teamID, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		storeID := rs.Primary.Attributes["store_id"]
		pathname := rs.Primary.Attributes["pathname"]
		if storeID == "" || pathname == "" {
			return fmt.Errorf("store_id or pathname is not set")
		}

		_, err := testClient.GetBlobObject(context.TODO(), client.GetBlobObjectRequest{
			Pathname: pathname,
			StoreID:  storeID,
			TeamID:   teamID,
		})
		return err
	}
}

func testCheckBlobObjectDeleted(testClient *client.Client, resourceName, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		storeID := rs.Primary.Attributes["store_id"]
		pathname := rs.Primary.Attributes["pathname"]
		if storeID == "" || pathname == "" {
			return fmt.Errorf("store_id or pathname is not set")
		}

		_, err := testClient.GetBlobObject(context.TODO(), client.GetBlobObjectRequest{
			Pathname: pathname,
			StoreID:  storeID,
			TeamID:   teamID,
		})
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("unexpected error checking for deleted blob object: %s", err)
		}

		return nil
	}
}

func testBlobObjectSourcePath(t *testing.T, filename string) string {
	t.Helper()

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("could not resolve working directory: %s", err)
	}

	return filepath.Join(workingDir, "testdata", "blob", filename)
}
