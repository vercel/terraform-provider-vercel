package vercel_test

import (
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAcc_DataSourceProjectDirectory(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectDirectoryConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_project_directory.test", "path", "examples/one"),
					testChecksum("data.vercel_project_directory.test", filepath.Join("files.examples", "one", "index.html"), Checksums{
						unix:    "60~9d3fedcc87ac72f54e75d4be7e06d0a6f8497e68",
						windows: "65~c0b8b91602dc7a394354cd9a21460ce2070b9a13",
					}),
					resource.TestCheckNoResourceAttr(
						"data.vercel_project_directory.test",
						filepath.Join("files.example", "file2.html"),
					),
					resource.TestCheckNoResourceAttr(
						"data.vercel_project_directory.test",
						filepath.Join("files.example", ".vercel", "output", "builds.json"),
					),
				),
			},
		},
	})
}

func testAccProjectDirectoryConfig() string {
	return `
data "vercel_project_directory" "test" {
    path = "examples/one"
}
`
}
