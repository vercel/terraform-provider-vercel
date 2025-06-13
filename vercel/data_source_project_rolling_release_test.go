package vercel_test

import (
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
			// First create the project and enable rolling release
			{
				Config: cfg(testAccProjectRollingReleasesConfigUpdate(nameSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectRollingReleaseExists(testClient(t), "vercel_project_rolling_release.example", testTeam(t)),
					resource.TestCheckResourceAttr("vercel_project_rolling_release.example", "rolling_release.enabled", "true"),
					resource.TestCheckResourceAttr("vercel_project_rolling_release.example", "rolling_release.advancement_type", "manual-approval"),
					resource.TestCheckResourceAttr("vercel_project_rolling_release.example", "rolling_release.stages.#", "4"),
				),
			},
			// Then disable it and check the data source
			{
				Config: cfg(testAccProjectRollingReleasesConfigOff(nameSuffix)),
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
