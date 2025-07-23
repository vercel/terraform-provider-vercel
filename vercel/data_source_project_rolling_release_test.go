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
				Config: cfg(testAccProjectRollingReleasesDataSourceConfig(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectRollingReleaseExists(testClient(t), "vercel_project_rolling_release.example", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project_rolling_release.example", "advancement_type", "manual-approval"),
					resource.TestCheckResourceAttr("vercel_project_rolling_release.example", "stages.#", "3"),
				),
			},
			{
				Config: cfg(testAccProjectRollingReleasesDataSourceConfigWithDataSource(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_project_rolling_release.example", "advancement_type", "manual-approval"),
					resource.TestCheckResourceAttr("data.vercel_project_rolling_release.example", "stages.#", "3"),
				),
			},
		},
	})
}

func testAccProjectRollingReleasesDataSourceConfig(nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%s"
	skew_protection = "12 hours"
}

resource "vercel_project_rolling_release" "example" {
	project_id = vercel_project.example.id
	advancement_type = "manual-approval"
	stages = [
		{
			target_percentage = 20
		},
		{
			target_percentage = 50
		},
		{
			target_percentage = 100
		}
	]
}
`, nameSuffix)
}

func testAccProjectRollingReleasesDataSourceConfigWithDataSource(nameSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
	name = "test-acc-example-project-%s"
	skew_protection = "12 hours"
}

resource "vercel_project_rolling_release" "example" {
	project_id = vercel_project.example.id
	advancement_type = "manual-approval"
	stages = [
		{
			target_percentage = 20
		},
		{
			target_percentage = 50
		},
		{
			target_percentage = 100
		}
	]
}

data "vercel_project_rolling_release" "example" {
	project_id = vercel_project.example.id
}
`, nameSuffix)
}
