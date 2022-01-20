package vercel_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAcc_ProjectDirectoryDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectDirectoryConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_project_directory.test", "path", "example"),
					resource.TestCheckResourceAttr("data.vercel_project_directory.test", "files.example/index.html", "60~9d3fedcc87ac72f54e75d4be7e06d0a6f8497e68"),
					resource.TestCheckNoResourceAttr("data.vercel_project_directory.test", "files.example/file2.html"),
				),
			},
		},
	})
}

func testAccProjectDirectoryConfig() string {
	return `
data "vercel_project_directory" "test" {
    path = "example"
}
`
}
