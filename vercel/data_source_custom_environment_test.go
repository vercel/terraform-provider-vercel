package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_CustomEnvironmentDataSource(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy("vercel_project.test", testTeam()),
		Steps: []resource.TestStep{
			{
				Config: testAccCustomEnvironmentDataSource(projectSuffix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.vercel_custom_environment.test", "id"),
					resource.TestCheckResourceAttrSet("data.vercel_custom_environment.test", "project_id"),
					resource.TestCheckResourceAttrSet("data.vercel_custom_environment.test", "name"),
					resource.TestCheckResourceAttr("data.vercel_custom_environment.test", "branch_tracking.type", "startsWith"),
					resource.TestCheckResourceAttr("data.vercel_custom_environment.test", "branch_tracking.pattern", "staging-"),
					resource.TestCheckResourceAttr("data.vercel_custom_environment.test", "description", "oh cool"),
				),
			},
		},
	})
}

func testAccCustomEnvironmentDataSource(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-custom-env-data-source-%[1]s"
  %[2]s
}

resource "vercel_custom_environment" "test" {
  project_id = vercel_project.test.id
  %[2]s
  name = "test-acc-custom-env-%[1]s"
  description = "oh cool"
  branch_tracking = {
    pattern = "staging-"
    type = "startsWith"
  }
}

data "vercel_custom_environment" "test" {
  project_id = vercel_project.test.id
  %[2]s
  name = "test-acc-custom-env-%[1]s"
}
`, projectSuffix, teamIDConfig())
}
