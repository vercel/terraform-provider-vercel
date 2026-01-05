package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

func testCheckLogDrainExists(testClient *client.Client, teamID, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetLogDrain(context.TODO(), rs.Primary.ID, teamID)
		return err
	}
}

func testCheckLogDrainDeleted(testClient *client.Client, n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetLogDrain(context.TODO(), rs.Primary.ID, teamID)
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("Unexpected error checking for deleted log drain: %s", err)
		}

		return nil
	}
}

func TestAcc_LogDrainResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckLogDrainDeleted(testClient(t), "vercel_log_drain.minimal", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccResourceLogDrain(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckLogDrainExists(testClient(t), testTeam(t), "vercel_log_drain.minimal"),
					resource.TestCheckResourceAttr("vercel_log_drain.minimal", "delivery_format", "json"),
					resource.TestCheckResourceAttr("vercel_log_drain.minimal", "environments.#", "1"),
					resource.TestCheckResourceAttr("vercel_log_drain.minimal", "environments.0", "production"),
					resource.TestCheckResourceAttr("vercel_log_drain.minimal", "sources.#", "1"),
					resource.TestCheckResourceAttr("vercel_log_drain.minimal", "sources.0", "static"),
					resource.TestCheckResourceAttrSet("vercel_log_drain.minimal", "endpoint"),
					resource.TestCheckResourceAttrSet("vercel_log_drain.minimal", "id"),
					resource.TestCheckResourceAttrSet("vercel_log_drain.minimal", "team_id"),
					resource.TestCheckResourceAttrSet("vercel_log_drain.maximal", "secret"),

					testCheckLogDrainExists(testClient(t), testTeam(t), "vercel_log_drain.maximal"),
					resource.TestCheckResourceAttr("vercel_log_drain.maximal", "delivery_format", "json"),
					resource.TestCheckResourceAttr("vercel_log_drain.maximal", "environments.#", "2"),
					resource.TestCheckResourceAttr("vercel_log_drain.maximal", "environments.0", "preview"),
					resource.TestCheckResourceAttr("vercel_log_drain.maximal", "environments.1", "production"),
					resource.TestCheckResourceAttr("vercel_log_drain.maximal", "sources.#", "7"),
					resource.TestCheckResourceAttr("vercel_log_drain.maximal", "sources.0", "build"),
					resource.TestCheckResourceAttr("vercel_log_drain.maximal", "sources.1", "edge"),
					resource.TestCheckResourceAttr("vercel_log_drain.maximal", "sources.2", "external"),
					resource.TestCheckResourceAttr("vercel_log_drain.maximal", "sources.3", "firewall"),
					resource.TestCheckResourceAttr("vercel_log_drain.maximal", "sources.4", "lambda"),
					resource.TestCheckResourceAttr("vercel_log_drain.maximal", "sources.5", "redirect"),
					resource.TestCheckResourceAttr("vercel_log_drain.maximal", "sources.6", "static"),
					resource.TestCheckResourceAttr("vercel_log_drain.maximal", "secret", "a_very_long_and_very_well_specified_secret"),
					resource.TestCheckResourceAttr("vercel_log_drain.maximal", "headers.%", "1"),
					resource.TestCheckResourceAttrSet("vercel_log_drain.maximal", "endpoint"),
					resource.TestCheckResourceAttrSet("vercel_log_drain.maximal", "id"),
					resource.TestCheckResourceAttrSet("vercel_log_drain.maximal", "team_id"),
				),
			},
		},
	})
}

func testAccResourceLogDrain(name string) string {
	return fmt.Sprintf(`
data "vercel_endpoint_verification" "test" {
}

resource "vercel_log_drain" "minimal" {
    name                    = "minimal-%[1]s"
    delivery_format         = "json"
    environments            = ["production"]
    sources                 = ["static"]
    endpoint = "https://verify-test-rouge.vercel.app/api?${data.vercel_endpoint_verification.test.verification_code}"
}

resource "vercel_project" "test" {
    name = "test-acc-%[1]s"
}

resource "vercel_log_drain" "maximal" {
    name                    = "maximal-%[1]s"
    delivery_format         = "json"
    environments            = ["production", "preview"]
    headers                 = {
        some-key = "some-value"
    }
    project_ids             = [vercel_project.test.id]
    sampling_rate           = 0.8
    secret                  = "a_very_long_and_very_well_specified_secret"
    sources                 = ["static", "edge", "external", "build", "lambda", "firewall", "redirect"]
    endpoint = "https://verify-test-rouge.vercel.app/api?${data.vercel_endpoint_verification.test.verification_code}"
}
`, name)
}
