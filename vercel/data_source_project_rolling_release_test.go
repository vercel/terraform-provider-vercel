package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_ProjectRollingReleaseDataSource(t *testing.T) {
	nameSuffix := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccProjectDestroy(testClient(t), "vercel_project.example", testTeam(t)),
		),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectRollingReleasesConfigOnWithDataSource(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectRollingReleaseExists(testClient(t), "vercel_project_rolling_release.example", testTeam(t)),
					resource.TestCheckResourceAttr("data.vercel_project_rolling_release.example", "automatic_rolling_release.#", "1"),
					resource.TestCheckResourceAttr("data.vercel_project_rolling_release.example", "automatic_rolling_release.0.target_percentage", "10"),
					resource.TestCheckResourceAttr("data.vercel_project_rolling_release.example", "automatic_rolling_release.0.duration", "10"),
				),
			},
		},
	})
}

func testAccProjectRollingReleasesConfigOnWithDataSource(nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%s"
}

resource "vercel_project_rolling_release" "example" {
	project_id = vercel_project.example.id
	automatic_rolling_release = [{
		target_percentage = 10
		duration = 10
	}]
}

data "vercel_project_rolling_release" "example" {
	project_id = vercel_project_rolling_release.example.project_id
}
`, nameSuffix)
}
