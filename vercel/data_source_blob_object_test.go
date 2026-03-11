package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_BlobObjectDataSource(t *testing.T) {
	suffix := acctest.RandString(16)
	storeName := fmt.Sprintf("test-acc-blob-ds-%s", suffix)
	sourceOne := testBlobObjectSourcePath(t, "object-one.txt")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccBlobObjectDataSourceConfig(storeName, sourceOne)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.vercel_blob_object.test", "id", "vercel_blob_object.test", "id"),
					resource.TestCheckResourceAttrPair("data.vercel_blob_object.test", "store_id", "vercel_blob_object.test", "store_id"),
					resource.TestCheckResourceAttrPair("data.vercel_blob_object.test", "pathname", "vercel_blob_object.test", "pathname"),
					resource.TestCheckResourceAttrPair("data.vercel_blob_object.test", "url", "vercel_blob_object.test", "url"),
					resource.TestCheckResourceAttrPair("data.vercel_blob_object.test", "download_url", "vercel_blob_object.test", "download_url"),
					resource.TestCheckResourceAttrPair("data.vercel_blob_object.test", "content_type", "vercel_blob_object.test", "content_type"),
					resource.TestCheckResourceAttrPair("data.vercel_blob_object.test", "cache_control", "vercel_blob_object.test", "cache_control"),
					resource.TestCheckResourceAttrPair("data.vercel_blob_object.test", "cache_control_max_age", "vercel_blob_object.test", "cache_control_max_age"),
					resource.TestCheckResourceAttrPair("data.vercel_blob_object.test", "etag", "vercel_blob_object.test", "etag"),
					resource.TestCheckResourceAttr("data.vercel_blob_object.test", "team_id", testTeam(t)),
					resource.TestCheckResourceAttrSet("data.vercel_blob_object.test", "size"),
					resource.TestCheckResourceAttrSet("data.vercel_blob_object.test", "uploaded_at"),
					resource.TestCheckResourceAttrSet("data.vercel_blob_object.test", "content_disposition"),
				),
			},
		},
	})
}

func testAccBlobObjectDataSourceConfig(storeName, source string) string {
	return fmt.Sprintf(`
resource "vercel_blob_store" "test" {
  name = "%s"
}

resource "vercel_blob_object" "test" {
  store_id              = vercel_blob_store.test.id
  pathname              = "terraform/data-source.txt"
  source                = "%s"
  content_type          = "text/plain"
  cache_control_max_age = 3600
}

data "vercel_blob_object" "test" {
  store_id = vercel_blob_store.test.id
  pathname = vercel_blob_object.test.pathname
  depends_on = [
    vercel_blob_object.test,
  ]
}
`, storeName, source)
}
