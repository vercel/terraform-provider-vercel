package vercel_test

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

type Checksums struct {
	windows string
	unix    string
}

func testChecksum(n, attribute string, checksums Checksums) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		sizeAndChecksum := rs.Primary.Attributes[attribute]
		if runtime.GOOS == "windows" {
			if sizeAndChecksum != checksums.windows {
				return fmt.Errorf("attribute %s expected %s but got %s", attribute, checksums.unix, sizeAndChecksum)
			}

			return nil
		}

		if sizeAndChecksum != checksums.unix {
			return fmt.Errorf("attribute %s expected %s but got %s", attribute, checksums.unix, sizeAndChecksum)
		}

		return nil
	}
}

func TestAcc_DataSourceFile(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFileConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_file.test", "path", "examples/one/index.html"),
					resource.TestCheckResourceAttr("data.vercel_file.test", "id", "examples/one/index.html"),
					testChecksum("data.vercel_file.test", "file.examples/one/index.html", Checksums{
						unix:    "60~9d3fedcc87ac72f54e75d4be7e06d0a6f8497e68",
						windows: "65~c0b8b91602dc7a394354cd9a21460ce2070b9a13",
					}),
				),
			},
		},
	})
}

func testAccFileConfig() string {
	return `
data "vercel_file" "test" {
    path = "examples/one/index.html"
}
`
}
