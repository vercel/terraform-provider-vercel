package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_DeploymentDataSource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccDeploymentDataSourceConfig(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.vercel_deployment.by_id", "id"),
					resource.TestCheckResourceAttrSet("data.vercel_deployment.by_id", "project_id"),
					resource.TestCheckResourceAttrSet("data.vercel_deployment.by_id", "url"),
					resource.TestCheckResourceAttr("data.vercel_deployment.by_id", "production", "true"),
					resource.TestCheckResourceAttr("data.vercel_deployment.by_id", "domains.#", "2"),
					resource.TestCheckResourceAttr("data.vercel_deployment.by_id", "meta.build", "123"),
					resource.TestCheckResourceAttr("data.vercel_deployment.by_id", "meta.env", "staging"),

					resource.TestCheckResourceAttrSet("data.vercel_deployment.by_url", "id"),
					resource.TestCheckResourceAttrSet("data.vercel_deployment.by_url", "project_id"),
					resource.TestCheckResourceAttrSet("data.vercel_deployment.by_url", "url"),
					resource.TestCheckResourceAttr("data.vercel_deployment.by_url", "production", "true"),
					resource.TestCheckResourceAttr("data.vercel_deployment.by_url", "domains.#", "2"),
					resource.TestCheckResourceAttr("data.vercel_deployment.by_url", "meta.build", "123"),
					resource.TestCheckResourceAttr("data.vercel_deployment.by_url", "meta.env", "staging"),
				),
			}},
	})
}

func testAccDeploymentDataSourceConfig(name string) string {
	return fmt.Sprintf(`
data "vercel_deployment" "by_id" {
   id = vercel_deployment.test.id
}

data "vercel_deployment" "by_url" {
   id = vercel_deployment.test.url
}

resource "vercel_project" "test" {
  name = "test-acc-deployment-%[1]s"
  environment = [
    {
      key    = "bar"
      value  = "baz"
      target = ["preview"]
    }
  ]
}

data "vercel_prebuilt_project" "test" {
    path = "examples/two"
}

resource "vercel_deployment" "test" {
  project_id = vercel_project.test.id

  production  = true
  files       = data.vercel_prebuilt_project.test.output
  path_prefix = data.vercel_prebuilt_project.test.path

  meta = {
    build = "123"
    env   = "staging"
  }
}
`, name)
}

func TestAcc_DeploymentDataSourceWithCustomEnvironment(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccDeploymentDataSourceWithCustomEnvironmentConfig(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.vercel_deployment.test", "id"),
					resource.TestCheckResourceAttrSet("data.vercel_deployment.test", "project_id"),
					resource.TestCheckResourceAttrSet("data.vercel_deployment.test", "url"),
					resource.TestCheckResourceAttrSet("data.vercel_deployment.test", "custom_environment_id"),
					resource.TestCheckResourceAttrPair("data.vercel_deployment.test", "custom_environment_id", "vercel_custom_environment.test", "id"),
				),
			},
		},
	})
}

func testAccDeploymentDataSourceWithCustomEnvironmentConfig(name string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-deployment-ds-custom-env-%[1]s"
}

resource "vercel_custom_environment" "test" {
  project_id = vercel_project.test.id
  name = "test-custom-env-%[1]s"
  description = "test custom environment for deployment data source"
}

data "vercel_prebuilt_project" "test" {
    path = "examples/two"
}

resource "vercel_deployment" "test" {
  project_id = vercel_project.test.id
  custom_environment_id = vercel_custom_environment.test.id

  files       = data.vercel_prebuilt_project.test.output
  path_prefix = data.vercel_prebuilt_project.test.path
}

data "vercel_deployment" "test" {
   id = vercel_deployment.test.id
}
`, name)
}
