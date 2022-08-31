package vercel_test

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAcc_DataSourcePrebuiltProject(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      prebuiltProjectNoOutput(),
				ExpectError: regexp.MustCompile(strings.ReplaceAll(`A prebuilt project data source was used, but no prebuilt output was found in \x60.\x60.`, " ", `\s*`)),
			},
			{
				Config: prebuiltProjectFailedBuild(),
				ExpectError: regexp.MustCompile(
					strings.ReplaceAll(`The prebuilt deployment at \x60examples/one\x60 cannot be used because \x60vercel build\x60\s*failed with an error`, " ", `\s*`),
				),
			},
			{
				Config: prebuiltProjectValid(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_prebuilt_project.test", "path", "examples/two"),
					resource.TestCheckResourceAttr("data.vercel_prebuilt_project.test", "id", "examples/two"),
					testChecksum(
						"data.vercel_prebuilt_project.test",
						filepath.Join("output.examples", "two", ".vercel", "output", "config.json"),
						Checksums{
							unix:    "19~e963e8b508fbae85b362afd1cd388c251fa24eee",
							windows: "19~e963e8b508fbae85b362afd1cd388c251fa24eee",
						},
					),
				),
			},
		},
	})
}

func prebuiltProjectNoOutput() string {
	return `
data "vercel_prebuilt_project" "test" {
    path = "."
}
`
}

func prebuiltProjectFailedBuild() string {
	return `
data "vercel_prebuilt_project" "test" {
    path = "examples/one"
}
`
}

func prebuiltProjectValid() string {
	return `
data "vercel_prebuilt_project" "test" {
    path = "examples/two"
}`
}
