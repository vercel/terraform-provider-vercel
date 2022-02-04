package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAcc_ProjectDataSource(t *testing.T) {
	t.Parallel()
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_project.test", "name", name),
					resource.TestCheckResourceAttr("data.vercel_project.test", "build_command", "npm run build"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "dev_command", "npm run serve"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "framework", "create-react-app"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "install_command", "npm install"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "output_directory", ".output"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "public_source", "true"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "root_directory", "ui/src"),
					resource.TestCheckTypeSetElemNestedAttrs("data.vercel_project.test", "environment.*", map[string]string{
						"key":   "foo",
						"value": "bar",
					}),
					resource.TestCheckTypeSetElemAttr("data.vercel_project.test", "environment.0.target.*", "production"),
				),
			},
		},
	})
}

func testAccProjectDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "%s"
  build_command = "npm run build"
  dev_command = "npm run serve"
  framework = "create-react-app"
  install_command = "npm install"
  output_directory = ".output"
  public_source = true
  root_directory = "ui/src"
  environment = [
    {
      key    = "foo"
      value  = "bar"
      target = ["production"]
    }
  ]
}

data "vercel_project" "test" {
    name = vercel_project.test.name
}
`, name)
}
