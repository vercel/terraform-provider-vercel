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

func TestAcc_BlobProjectConnectionResource(t *testing.T) {
	suffix := acctest.RandString(16)
	storeName := fmt.Sprintf("test-acc-blob-%s", suffix)
	projectName := fmt.Sprintf("test-acc-blob-project-%s", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckBlobProjectConnectionDeleted(testClient(t), "vercel_blob_project_connection.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccBlobProjectConnectionResourceConfig(storeName, projectName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckBlobProjectConnectionExists(testClient(t), testTeam(t), "vercel_blob_project_connection.test"),
					resource.TestCheckResourceAttrPair("vercel_blob_project_connection.test", "blob_store_id", "vercel_blob_store.test", "id"),
					resource.TestCheckResourceAttrPair("vercel_blob_project_connection.test", "project_id", "vercel_project.test", "id"),
					resource.TestCheckResourceAttr("vercel_blob_project_connection.test", "team_id", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_blob_project_connection.test", "env_var_prefix", "BLOB"),
					resource.TestCheckResourceAttr("vercel_blob_project_connection.test", "read_write_token_env_var_name", "BLOB_READ_WRITE_TOKEN"),
					resource.TestCheckResourceAttr("vercel_blob_project_connection.test", "environments.#", "2"),
					resource.TestCheckTypeSetElemAttr("vercel_blob_project_connection.test", "environments.*", "preview"),
					resource.TestCheckTypeSetElemAttr("vercel_blob_project_connection.test", "environments.*", "production"),
					resource.TestCheckResourceAttrSet("vercel_blob_project_connection.test", "id"),
				),
			},
			{
				ResourceName:      "vercel_blob_project_connection.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getBlobProjectConnectionImportID("vercel_blob_project_connection.test"),
			},
			{
				Config: cfg(testAccBlobProjectConnectionResourceUpdatedConfig(storeName, projectName)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("vercel_blob_project_connection.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckBlobProjectConnectionExists(testClient(t), testTeam(t), "vercel_blob_project_connection.test"),
					resource.TestCheckResourceAttr("vercel_blob_project_connection.test", "env_var_prefix", "ASSETS"),
					resource.TestCheckResourceAttr("vercel_blob_project_connection.test", "read_write_token_env_var_name", "ASSETS_READ_WRITE_TOKEN"),
					resource.TestCheckResourceAttr("vercel_blob_project_connection.test", "environments.#", "2"),
					resource.TestCheckTypeSetElemAttr("vercel_blob_project_connection.test", "environments.*", "development"),
					resource.TestCheckTypeSetElemAttr("vercel_blob_project_connection.test", "environments.*", "preview"),
				),
			},
		},
	})
}

func testAccBlobProjectConnectionResourceConfig(storeName, projectName string) string {
	return fmt.Sprintf(`
resource "vercel_blob_store" "test" {
  name = "%s"
}

resource "vercel_project" "test" {
  name = "%s"
}

resource "vercel_blob_project_connection" "test" {
  blob_store_id = vercel_blob_store.test.id
  project_id    = vercel_project.test.id
}
`, storeName, projectName)
}

func testAccBlobProjectConnectionResourceUpdatedConfig(storeName, projectName string) string {
	return fmt.Sprintf(`
resource "vercel_blob_store" "test" {
  name = "%s"
}

resource "vercel_project" "test" {
  name = "%s"
}

resource "vercel_blob_project_connection" "test" {
  blob_store_id   = vercel_blob_store.test.id
  project_id      = vercel_project.test.id
  env_var_prefix  = "ASSETS"
  environments    = ["development", "preview"]
}
`, storeName, projectName)
}

func getBlobProjectConnectionImportID(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("not found: %s", resourceName)
		}

		connectionID := rs.Primary.ID
		if connectionID == "" {
			return "", fmt.Errorf("no ID is set")
		}

		storeID := rs.Primary.Attributes["blob_store_id"]
		if storeID == "" {
			return "", fmt.Errorf("no blob_store_id is set")
		}

		teamID := rs.Primary.Attributes["team_id"]
		if teamID == "" {
			return fmt.Sprintf("%s/%s", storeID, connectionID), nil
		}

		return fmt.Sprintf("%s/%s/%s", teamID, storeID, connectionID), nil
	}
}

func testCheckBlobProjectConnectionExists(testClient *client.Client, teamID, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		storeID := rs.Primary.Attributes["blob_store_id"]
		if storeID == "" {
			return fmt.Errorf("no blob_store_id is set")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetBlobStoreConnection(context.TODO(), storeID, rs.Primary.ID, teamID)
		return err
	}
}

func testCheckBlobProjectConnectionDeleted(testClient *client.Client, resourceName, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		storeID := rs.Primary.Attributes["blob_store_id"]
		if storeID == "" {
			return fmt.Errorf("no blob_store_id is set")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetBlobStoreConnection(context.TODO(), storeID, rs.Primary.ID, teamID)
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("unexpected error checking for deleted blob store project connection: %s", err)
		}

		return nil
	}
}
