package vercel_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAcc_FileDataSource(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFileConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_file.test", "path", "example/index.html"),
					resource.TestCheckResourceAttr("data.vercel_file.test", "id", "example/index.html"),
					resource.TestCheckResourceAttr("data.vercel_file.test", "file.example/index.html", "60~9d3fedcc87ac72f54e75d4be7e06d0a6f8497e68"),
				),
			},
		},
	})
}

func testAccFileConfig() string {
	return `
data "vercel_file" "test" {
    path = "example/index.html"
}
`
}
