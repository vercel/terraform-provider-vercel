package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func testCheckTraceDrainExists(testClient *client.Client, teamID, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetTraceDrain(context.TODO(), rs.Primary.ID, teamID)
		return err
	}
}

func testCheckTraceDrainDeleted(testClient *client.Client, n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetTraceDrain(context.TODO(), rs.Primary.ID, teamID)
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("unexpected error checking for deleted trace drain: %s", err)
		}

		return nil
	}
}

func TestAcc_TraceDrainResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckTraceDrainDeleted(testClient(t), "vercel_trace_drain.minimal", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccResourceTraceDrain(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTraceDrainExists(testClient(t), testTeam(t), "vercel_trace_drain.minimal"),
					resource.TestCheckResourceAttr("vercel_trace_drain.minimal", "delivery_format", "json"),
					resource.TestCheckResourceAttr("vercel_trace_drain.minimal", "endpoint", "https://verify-test-rouge.vercel.app/api"),
					resource.TestCheckResourceAttrSet("vercel_trace_drain.minimal", "id"),
					resource.TestCheckResourceAttrSet("vercel_trace_drain.minimal", "team_id"),

					testCheckTraceDrainExists(testClient(t), testTeam(t), "vercel_trace_drain.maximal"),
					resource.TestCheckResourceAttr("vercel_trace_drain.maximal", "delivery_format", "json"),
					resource.TestCheckResourceAttr("vercel_trace_drain.maximal", "headers.%", "1"),
					resource.TestCheckResourceAttr("vercel_trace_drain.maximal", "sampling_rules.#", "1"),
					resource.TestCheckResourceAttr("vercel_trace_drain.maximal", "sampling_rules.0.rate", "0.8"),
					resource.TestCheckResourceAttr("vercel_trace_drain.maximal", "sampling_rules.0.environment", "production"),
					resource.TestCheckResourceAttr("vercel_trace_drain.maximal", "sampling_rules.0.request_path", "/api"),
					resource.TestCheckResourceAttr("vercel_trace_drain.maximal", "secret", "a_very_long_and_very_well_specified_secret"),
					resource.TestCheckResourceAttrSet("vercel_trace_drain.maximal", "id"),
					resource.TestCheckResourceAttrSet("vercel_trace_drain.maximal", "team_id"),
				),
			},
		},
	})
}

func testAccResourceTraceDrain(name string) string {
	return fmt.Sprintf(`
resource "vercel_trace_drain" "minimal" {
    name                    = "minimal-%[1]s"
    delivery_format         = "json"
    endpoint                = "https://verify-test-rouge.vercel.app/api"
}

resource "vercel_project" "test" {
    name = "test-acc-%[1]s"
}

resource "vercel_trace_drain" "maximal" {
    name                    = "maximal-%[1]s"
    delivery_format         = "json"
    headers                 = {
        some-key = "some-value"
    }
    project_ids             = [vercel_project.test.id]
    sampling_rules          = [{
        rate         = 0.8
        environment  = "production"
        request_path = "/api"
    }]
    secret                  = "a_very_long_and_very_well_specified_secret"
    endpoint                = "https://verify-test-rouge.vercel.app/api"
}
`, name)
}
