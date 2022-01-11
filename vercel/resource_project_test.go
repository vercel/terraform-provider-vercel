package vercel_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/vercel/terraform-provider-vercel/client"
)

func testAccVercelProjectExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no projectID is set")
		}

		c := client.New(os.Getenv("VERCEL_API_TOKEN"))
		_, err := c.GetProject(context.TODO(), rs.Primary.ID, "")
		return err
	}
}

func TestAccVercelProject(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccProject("one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccVercelProjectExists("vercel_project.test"),
					resource.TestCheckResourceAttr("vercel_project.test", "name", "test-acc-one"),
					resource.TestCheckResourceAttr("vercel_project.test", "build_command", "npm run build"),
					resource.TestCheckResourceAttr("vercel_project.test", "dev_command", "npm run serve"),
					resource.TestCheckResourceAttr("vercel_project.test", "framework", "create-react-app"),
					resource.TestCheckResourceAttr("vercel_project.test", "install_command", "npm install"),
					resource.TestCheckResourceAttr("vercel_project.test", "output_directory", ".output"),
					resource.TestCheckResourceAttr("vercel_project.test", "public_source", "true"),
					resource.TestCheckResourceAttr("vercel_project.test", "root_directory", "src"),
					resource.TestCheckTypeSetElemNestedAttrs("vercel_project.test", "environment.*", map[string]string{
						"key":   "foo",
						"value": "bar",
					}),
					resource.TestCheckTypeSetElemAttr("vercel_project.test", "environment.0.target.*", "production"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "vercel_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update testing
			{
				Config: testAccProjectUpdated("two"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project.test", "name", "test-acc-two"),
					resource.TestCheckNoResourceAttr("vercel_project.test", "build_command"),
					resource.TestCheckTypeSetElemNestedAttrs("vercel_project.test", "environment.*", map[string]string{
						"key":   "bar",
						"value": "baz",
					}),
				),
			},
		},
	})
}

func testAccProjectUpdated(name string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-%s"
  environment = [
    {
      key    = "bar"
      value  = "baz"
      target = ["preview"]
    }
  ]
}
`, name)
}

func testAccProject(name string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-%s"
  build_command = "npm run build"
  dev_command = "npm run serve"
  framework = "create-react-app"
  install_command = "npm install"
  output_directory = ".output"
  public_source = true
  root_directory = "src"
  environment = [
    {
      key    = "foo"
      value  = "bar"
      target = ["production"]
    }
  ]
}
`, name)
}
