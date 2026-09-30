package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_EnvironmentDataSource(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccEnvironmentDataSource(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_environment.preview", "id", "preview"),
					resource.TestCheckResourceAttr("data.vercel_environment.preview", "slug", "preview"),
					resource.TestCheckResourceAttr("data.vercel_environment.preview", "kind", "system"),
					resource.TestCheckResourceAttr("data.vercel_environment.preview", "managed_by", "vercel"),
					resource.TestCheckResourceAttr("data.vercel_environment.preview", "type", "preview"),
					resource.TestCheckResourceAttr("data.vercel_environment.preview", "capabilities.manage", "false"),
					resource.TestCheckResourceAttr("data.vercel_environment.preview", "capabilities.domains", "true"),
					resource.TestCheckResourceAttr("data.vercel_environment.preview", "capabilities.deployments", "true"),
					resource.TestCheckResourceAttrSet("data.vercel_environment.custom", "id"),
					resource.TestCheckResourceAttr("data.vercel_environment.custom", "slug", "staging"),
					resource.TestCheckResourceAttr("data.vercel_environment.custom", "kind", "custom"),
					resource.TestCheckResourceAttr("data.vercel_environment.custom", "managed_by", "user"),
					resource.TestCheckResourceAttr("data.vercel_environment.custom", "capabilities.manage", "true"),
					resource.TestCheckResourceAttr("data.vercel_environment.custom", "branch_tracking.type", "startsWith"),
					resource.TestCheckResourceAttr("data.vercel_environment.custom", "branch_tracking.pattern", "staging-"),
					resource.TestCheckResourceAttrPair("data.vercel_environment.custom", "id", "vercel_custom_environment.test", "id"),
				),
			},
		},
	})
}

func testAccEnvironmentDataSource(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-environment-ds-%[1]s"
}

resource "vercel_custom_environment" "test" {
  project_id  = vercel_project.test.id
  name        = "staging"
  description = "QA staging"
  branch_tracking = {
    pattern = "staging-"
    type    = "startsWith"
  }
}

data "vercel_environment" "preview" {
  project_id = vercel_project.test.id
  slug       = "preview"
}

data "vercel_environment" "custom" {
  project_id = vercel_project.test.id
  id         = vercel_custom_environment.test.id
}
`, projectSuffix)
}
