package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAcc_ProjectFunctionMaxDurationResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectFunctionMaxDurationResourceConfig(name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_function_max_duration.elevated", "max_duration", "elevated"),
				),
			},
			{
				ImportState:  true,
				ResourceName: "vercel_project_function_max_duration.elevated",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["vercel_project_function_max_duration.elevated"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.ID), nil
				},
			},
			{
				Config: testAccProjectFunctionMaxDurationResourceConfigUpdated(name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_function_max_duration.elevated", "max_duration", "performance"),
				),
			},
		},
	})
}

func testAccProjectFunctionMaxDurationResourceConfig(name, teamID string) string {
	return fmt.Sprintf(`
resource "vercel_project" "elevated" {
    name = "test-acc-%[1]s"
    %[2]s
}

resource "vercel_project_function_max_duration" "elevated" {
    project_id = vercel_project.elevated.id
    max_duration = "100"
    %[2]s
}
`, name, teamID)
}

func testAccProjectFunctionMaxDurationResourceConfigUpdated(name, teamID string) string {
	return fmt.Sprintf(`
resource "vercel_project" "elevated" {
    name = "test-acc-%[1]s"
    %[2]s
}

resource "vercel_project_function_max_duration" "elevated" {
    project_id = vercel_project.elevated.id
    max_duration = 100
    %[2]s
}
`, name, teamID)
}
