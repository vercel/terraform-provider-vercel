package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_ProjectDataSource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectDataSourceConfig(name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_project.test", "name", "test-acc-"+name),
					resource.TestCheckResourceAttr("data.vercel_project.test", "build_command", "npm run build"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "dev_command", "npm run serve"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "framework", "nextjs"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "install_command", "npm install"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "output_directory", ".output"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "public_source", "true"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "root_directory", "ui/src"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "vercel_authentication.deployment_type", "standard_protection"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "password_protection.deployment_type", "standard_protection"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "trusted_ips.addresses.#", "1"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "automatically_expose_system_environment_variables", "true"),
					resource.TestCheckTypeSetElemNestedAttrs("data.vercel_project.test", "trusted_ips.addresses.*", map[string]string{
						"value": "1.1.1.1",
						"note":  "notey note",
					}),
					resource.TestCheckResourceAttr("data.vercel_project.test", "trusted_ips.deployment_type", "only_production_deployments"),
					resource.TestCheckResourceAttr("data.vercel_project.test", "trusted_ips.protection_mode", "trusted_ip_required"),

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

func testAccProjectDataSourceConfig(name, teamID string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-%s"
  build_command = "npm run build"
  dev_command = "npm run serve"
  framework = "nextjs"
  install_command = "npm install"
  output_directory = ".output"
  public_source = true
  root_directory = "ui/src"
  vercel_authentication = {
    deployment_type = "standard_protection"
  }
  password_protection = {
    password = "foo"
    deployment_type = "standard_protection"
  }
  trusted_ips = {
	addresses = [
		{
			value = "1.1.1.1"
			note = "notey note"
		}
	]
	deployment_type = "only_production_deployments"
	protection_mode = "trusted_ip_required"
  }
  %s
  environment = [
    {
      key    = "foo"
      value  = "bar"
      target = ["production"]
    }
  ]
  automatically_expose_system_environment_variables = true
}

data "vercel_project" "test" {
    name = vercel_project.test.name
    %[2]s
}
`, name, teamID)
}
