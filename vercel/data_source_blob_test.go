package vercel_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAcc_BlobDataSources(t *testing.T) {
	suffix := acctest.RandString(16)
	storeName := fmt.Sprintf("test-acc-blob-%s", suffix)
	projectName := fmt.Sprintf("test-acc-blob-project-%s", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccBlobDataSourcesConfig(storeName, projectName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.vercel_blob_store.test", "id", "vercel_blob_store.test", "id"),
					resource.TestCheckResourceAttr("data.vercel_blob_store.test", "name", storeName),
					resource.TestCheckResourceAttr("data.vercel_blob_store.test", "access", "public"),
					resource.TestCheckResourceAttr("data.vercel_blob_store.test", "region", "iad1"),
					resource.TestCheckResourceAttr("data.vercel_blob_store.test", "team_id", testTeam(t)),
					resource.TestCheckResourceAttrSet("data.vercel_blob_store.test", "file_count"),
					testCheckBlobStoreListed("data.vercel_blob_stores.test", "vercel_blob_store.test"),
					resource.TestCheckResourceAttr("data.vercel_blob_stores.test", "team_id", testTeam(t)),
					resource.TestCheckResourceAttrPair("data.vercel_blob_project_connections.test", "store_id", "vercel_blob_store.test", "id"),
					resource.TestCheckResourceAttr("data.vercel_blob_project_connections.test", "connections.#", "1"),
					resource.TestCheckResourceAttrPair("data.vercel_blob_project_connections.test", "connections.0.project_id", "vercel_project.test", "id"),
					resource.TestCheckResourceAttr("data.vercel_blob_project_connections.test", "connections.0.project_name", projectName),
					resource.TestCheckResourceAttr("data.vercel_blob_project_connections.test", "connections.0.env_var_prefix", "ASSETS"),
					resource.TestCheckResourceAttr("data.vercel_blob_project_connections.test", "connections.0.read_write_token_env_var_name", "ASSETS_READ_WRITE_TOKEN"),
					resource.TestCheckResourceAttr("data.vercel_blob_project_connections.test", "connections.0.environments.#", "1"),
					resource.TestCheckTypeSetElemAttr("data.vercel_blob_project_connections.test", "connections.0.environments.*", "preview"),
				),
			},
		},
	})
}

func testAccBlobDataSourcesConfig(storeName, projectName string) string {
	return fmt.Sprintf(`
resource "vercel_blob_store" "test" {
  name = "%s"
}

resource "vercel_project" "test" {
  name = "%s"
}

resource "vercel_blob_project_connection" "test" {
  blob_store_id  = vercel_blob_store.test.id
  project_id     = vercel_project.test.id
  env_var_prefix = "ASSETS"
  environments   = ["preview"]
}

data "vercel_blob_store" "test" {
  id = vercel_blob_store.test.id
}

data "vercel_blob_stores" "test" {
  depends_on = [
    vercel_blob_store.test,
  ]
}

data "vercel_blob_project_connections" "test" {
  store_id = vercel_blob_store.test.id
  depends_on = [
    vercel_blob_project_connection.test,
  ]
}

`, storeName, projectName)
}

func testCheckBlobStoreListed(dataSourceName, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		targetID := resourceState.Primary.ID
		if targetID == "" {
			return fmt.Errorf("no ID is set for %s", resourceName)
		}

		dataSourceState, ok := s.RootModule().Resources[dataSourceName]
		if !ok {
			return fmt.Errorf("not found: %s", dataSourceName)
		}

		storeCount, err := strconv.Atoi(dataSourceState.Primary.Attributes["stores.#"])
		if err != nil {
			return fmt.Errorf("invalid stores count: %w", err)
		}

		for index := 0; index < storeCount; index++ {
			if dataSourceState.Primary.Attributes[fmt.Sprintf("stores.%d.id", index)] == targetID {
				return nil
			}
		}

		return fmt.Errorf("blob store %s not found in %s", targetID, dataSourceName)
	}
}
