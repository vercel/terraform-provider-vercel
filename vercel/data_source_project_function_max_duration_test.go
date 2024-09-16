package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_ProjectFunctionMaxDurationDataSource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectFunctionMaxDurationDataSourceConfig(name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_project_function_max_duration.elevated", "max_duration", "100"),
				),
			},
		},
	})
}

func testAccProjectFunctionMaxDurationDataSourceConfig(name, teamID string) string {
	return fmt.Sprintf(`
resource "vercel_project" "elevated" {
    name = "test-acc-%[1]s"
    %[2]s
}

resource "vercel_project_function_cpu" "elevated" {
    project_id = vercel_project.elevated.id
    max_duration = 100
    %[2]s
}

data "vercel_project_function_cpu" "elevated" {
    project_id = vercel_project_function_cpu.elevated.project_id
    %[2]s
}
`, name, teamID)
}
