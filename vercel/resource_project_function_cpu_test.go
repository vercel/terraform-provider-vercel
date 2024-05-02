package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAcc_ProjectFunctionCPUResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectFunctionCPUResourceConfig(name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_function_cpu.basic", "cpu", "basic"),
					resource.TestCheckResourceAttr("vercel_project_function_cpu.standard", "cpu", "standard"),
					resource.TestCheckResourceAttr("vercel_project_function_cpu.performance", "cpu", "performance"),
				),
			},
			{
				ImportState:  true,
				ResourceName: "vercel_project_function_cpu.basic",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["vercel_project_function_cpu.basic"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.ID), nil
				},
			},
			{
				ImportState:  true,
				ResourceName: "vercel_project_function_cpu.standard",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["vercel_project_function_cpu.standard"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.ID), nil
				},
			},
			{
				ImportState:  true,
				ResourceName: "vercel_project_function_cpu.performance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["vercel_project_function_cpu.performance"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.ID), nil
				},
			},
			{
				Config: testAccProjectFunctionCPUResourceConfigUpdated(name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_function_cpu.basic", "cpu", "performance"),
				),
			},
		},
	})
}

func testAccProjectFunctionCPUResourceConfig(name, teamID string) string {
	return fmt.Sprintf(`
resource "vercel_project" "basic" {
    name = "test-acc-%[1]s"
    %[2]s
}

resource "vercel_project_function_cpu" "basic" {
    project_id = vercel_project.basic.id
    cpu = "basic"
    %[2]s
}

resource "vercel_project" "standard" {
    name = "test-acc-%[1]s-standard"
    %[2]s
}

resource "vercel_project_function_cpu" "standard" {
    project_id = vercel_project.standard.id
    cpu = "standard"
    %[2]s
}

resource "vercel_project" "performance" {
    name = "test-acc-%[1]s-performance"
    %[2]s
}

resource "vercel_project_function_cpu" "performance" {
    project_id = vercel_project.performance.id
    cpu = "performance"
    %[2]s
}
`, name, teamID)
}

func testAccProjectFunctionCPUResourceConfigUpdated(name, teamID string) string {
	return fmt.Sprintf(`
resource "vercel_project" "basic" {
    name = "test-acc-%[1]s"
    %[2]s
}

resource "vercel_project_function_cpu" "basic" {
    project_id = vercel_project.basic.id
    cpu = "performance"
    %[2]s
}
`, name, teamID)
}
