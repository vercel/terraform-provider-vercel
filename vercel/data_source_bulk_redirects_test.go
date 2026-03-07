package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_BulkRedirectsDataSource(t *testing.T) {
	nameSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy(testClient(t), "vercel_project.example", testTeam(t)),
		),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccBulkRedirectsDataSourceConfig(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccBulkRedirectsExists(testClient(t), "vercel_bulk_redirects.example", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_bulk_redirects.example", "redirects.#", "2"),
				),
			},
			{
				Config: cfg(testAccBulkRedirectsDataSourceConfigWithDataSource(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_bulk_redirects.example", "redirects.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("data.vercel_bulk_redirects.example", "redirects.*", map[string]string{
						"source":         "/old-path",
						"destination":    "/new-path",
						"status_code":    "307",
						"case_sensitive": "false",
						"query":          "false",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.vercel_bulk_redirects.example", "redirects.*", map[string]string{
						"source":         "/blog",
						"destination":    "https://example.com/blog",
						"status_code":    "308",
						"case_sensitive": "true",
						"query":          "true",
					}),
					resource.TestCheckResourceAttrSet("data.vercel_bulk_redirects.example", "version_id"),
					resource.TestCheckResourceAttr("data.vercel_bulk_redirects.by_version", "redirects.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("data.vercel_bulk_redirects.by_version", "redirects.*", map[string]string{
						"source":         "/old-path",
						"destination":    "/new-path",
						"status_code":    "307",
						"case_sensitive": "false",
						"query":          "false",
					}),
					resource.TestCheckResourceAttrSet("data.vercel_bulk_redirects.by_version", "version_id"),
				),
			},
		},
	})
}

func testAccBulkRedirectsDataSourceConfig(projectName string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%s"
}

resource "vercel_bulk_redirects" "example" {
	project_id = vercel_project.example.id
	redirects = [
		{
			source                = "/old-path"
			destination           = "/new-path"
			status_code           = 307
			case_sensitive        = false
			query                 = false
		},
		{
			source                = "/blog"
			destination           = "https://example.com/blog"
			status_code           = 308
			case_sensitive        = true
			query                 = true
		},
	]
}
`, projectName)
}

func testAccBulkRedirectsDataSourceConfigWithDataSource(projectName string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%s"
}

resource "vercel_bulk_redirects" "example" {
	project_id = vercel_project.example.id
	redirects = [
		{
			source                = "/old-path"
			destination           = "/new-path"
			status_code           = 307
			case_sensitive        = false
			query                 = false
		},
		{
			source                = "/blog"
			destination           = "https://example.com/blog"
			status_code           = 308
			case_sensitive        = true
			query                 = true
		},
	]
}

data "vercel_bulk_redirects" "example" {
	project_id = vercel_project.example.id
}

data "vercel_bulk_redirects" "by_version" {
	project_id = vercel_project.example.id
	version_id = vercel_bulk_redirects.example.version_id
}
`, projectName)
}
