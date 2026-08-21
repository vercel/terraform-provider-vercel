package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAcc_ProjectTracing(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy(testClient(t), "vercel_project.example", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccProjectTracingConfig(name, 0.25, "production")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_tracing.example", "sampling_rules.#", "1"),
					resource.TestCheckResourceAttr("vercel_project_tracing.example", "sampling_rules.0.rate", "0.25"),
					resource.TestCheckResourceAttr("vercel_project_tracing.example", "sampling_rules.0.environment", "production"),
					resource.TestCheckResourceAttr("vercel_project_tracing.example", "sampling_rules.0.request_path", "/api"),
				),
			},
			{
				ResourceName:      "vercel_project_tracing.example",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceState, ok := state.RootModule().Resources["vercel_project_tracing.example"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return fmt.Sprintf("%s/%s", resourceState.Primary.Attributes["team_id"], resourceState.Primary.Attributes["project_id"]), nil
				},
			},
			{
				Config: cfg(testAccProjectTracingConfig(name, 0.5, "preview")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_tracing.example", "sampling_rules.0.rate", "0.5"),
					resource.TestCheckResourceAttr("vercel_project_tracing.example", "sampling_rules.0.environment", "preview"),
				),
			},
		},
	})
}

func testAccProjectTracingConfig(name string, rate float64, environment string) string {
	return fmt.Sprintf(`
resource "vercel_project" "example" {
  name = "test-acc-project-tracing-%[1]s"
}

resource "vercel_project_tracing" "example" {
  project_id = vercel_project.example.id

  sampling_rules = [{
    rate         = %[2]g
    environment  = "%[3]s"
    request_path = "/api"
  }]
}
`, name, rate, environment)
}
