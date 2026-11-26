package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAcc_ProjectSpeedInsights(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.example", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectSpeedInsightsConfig(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("vercel_project_speed_insights.example", "project_id", "vercel_project.example", "id"),
					resource.TestCheckResourceAttrPair("vercel_project_speed_insights.example", "id", "vercel_project.example", "id"),
				),
			},
			{
				ResourceName:      "vercel_project_speed_insights.example",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceState, ok := state.RootModule().Resources["vercel_project_speed_insights.example"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return fmt.Sprintf("%s/%s", resourceState.Primary.Attributes["team_id"], resourceState.Primary.Attributes["project_id"]), nil
				},
			},
		},
	})
}

func testAccProjectSpeedInsightsConfig(name string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
  name = "test-acc-project-speed-insights-%[1]s"
}

resource "vercel_project_speed_insights" "example" {
  project_id = vercel_project.example.id
}
`, name)
}
