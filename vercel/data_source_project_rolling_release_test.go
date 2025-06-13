package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_ProjectRollingReleaseDataSource(t *testing.T) {
	return
	nameSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy(testClient(t), "vercel_project.example", testTeam(t)),
		),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectRollingReleasesConfigOffWithDataSource(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectRollingReleaseExists(testClient(t), "vercel_project_rolling_release.example", testTeam(t)),
					resource.TestCheckResourceAttr("data.vercel_project_rolling_release.example", "rolling_release.enabled", "false"),
					resource.TestCheckResourceAttr("data.vercel_project_rolling_release.example", "rolling_release.advancement_type", ""),
					resource.TestCheckResourceAttr("data.vercel_project_rolling_release.example", "rolling_release.stages.#", "0"),
				),
			},
		},
	})
}

func testAccProjectRollingReleasesConfigOffWithDataSource(nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%s"
}

resource "vercel_project_rolling_release" "example" {
	project_id = vercel_project.example.id
	rolling_release = {
		enabled          = false
		advancement_type = ""
		stages          = []
	}
}

data "vercel_project_rolling_release" "example" {
	project_id = vercel_project_rolling_release.example.project_id
}
`, nameSuffix)
}
