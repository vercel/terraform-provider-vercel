package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_ProjectFunctionCPUDataSource(t *testing.T) {
	t.Skip("the resource is deprecated and tests should be removed in the next release")
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectFunctionCPUDataSourceConfig(name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_project_function_cpu.basic", "cpu", "basic"),
					resource.TestCheckResourceAttr("data.vercel_project_function_cpu.standard", "cpu", "standard"),
					resource.TestCheckResourceAttr("data.vercel_project_function_cpu.performance", "cpu", "performance"),
				),
			},
		},
	})
}

func testAccProjectFunctionCPUDataSourceConfig(name, teamID string) string {
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

data "vercel_project_function_cpu" "basic" {
    project_id = vercel_project_function_cpu.basic.project_id
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

data "vercel_project_function_cpu" "standard" {
    project_id = vercel_project_function_cpu.standard.project_id
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

data "vercel_project_function_cpu" "performance" {
    project_id = vercel_project_function_cpu.performance.project_id
    %[2]s
}

`, name, teamID)
}
