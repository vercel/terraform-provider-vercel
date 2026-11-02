package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v5/client"
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

func TestAcc_LogDrainResource_WriteOnly(t *testing.T) {
	name := "write-only-" + acctest.RandString(16)
	const address = "vercel_log_drain.write_only"
	const initial = "test-log-drain-initial-secret-at-least-thirty-two-characters"
	const rotated = "test-log-drain-rotated-secret-at-least-thirty-two-characters"
	const legacy = "test-log-drain-legacy-secret-at-least-thirty-two-characters"
	t.Setenv("TF_VAR_log_drain_secret", initial)

	config := func(secretFields string) string {
		return cfg(fmt.Sprintf(`
variable "log_drain_secret" {
  type      = string
  sensitive = true
  ephemeral = true
}

data "vercel_endpoint_verification" "test" {}

resource "vercel_log_drain" "write_only" {
  name            = %q
  delivery_format = "ndjson"
  environments    = ["production"]
  sources         = ["static"]
  endpoint        = "https://verify-test-rouge.vercel.app/api?${data.vercel_endpoint_verification.test.verification_code}"
  %s
}
`, name, secretFields))
	}
	writeOnly := config("secret_wo = var.log_drain_secret")
	checkWriteOnly := func(secret string) resource.TestCheckFunc {
		return resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckNoResourceAttr(address, "secret"),
			resource.TestCheckNoResourceAttr(address, "secret_wo"),
			func(s *terraform.State) error {
				remote, err := testClient(t).GetLogDrain(context.Background(), s.RootModule().Resources[address].Primary.ID, testTeam(t))
				if err != nil {
					return err
				}
				if remote.Secret != secret {
					return fmt.Errorf("remote Log Drain has an unexpected signing secret")
				}
				return nil
			},
		)
	}
	replace := resource.ConfigPlanChecks{
		PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionReplace)},
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckLogDrainDeleted(testClient(t), address, testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: writeOnly,
				Check: resource.ComposeAggregateTestCheckFunc(
					checkWriteOnly(initial),
					resource.TestCheckNoResourceAttr(address, "secret_wo_version"),
				),
			},
			{
				PreConfig: func() { t.Setenv("TF_VAR_log_drain_secret", rotated) },
				Config:    writeOnly,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionNoop)},
				},
				Check: checkWriteOnly(initial),
			},
			{
				Config:           config("secret_wo = var.log_drain_secret\nsecret_wo_version = 0"),
				ConfigPlanChecks: replace,
				Check:            checkWriteOnly(rotated),
			},
			{
				Config:           config("secret_wo = var.log_drain_secret\nsecret_wo_version = 1"),
				ConfigPlanChecks: replace,
				Check:            checkWriteOnly(rotated),
			},
			{
				Config:           config(fmt.Sprintf("secret = %q", legacy)),
				ConfigPlanChecks: replace,
				Check:            resource.TestCheckResourceAttr(address, "secret", legacy),
			},
			{
				Config:           writeOnly,
				ConfigPlanChecks: replace,
				Check:            checkWriteOnly(rotated),
			},
			{
				ResourceName: address,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return "write-only:" + testTeam(t) + "/" + s.RootModule().Resources[address].Primary.ID, nil
				},
				ImportStateVerify: true,
			},
			{
				Config:           config(""),
				ConfigPlanChecks: replace,
				Check:            testCheckLogDrainExists(testClient(t), testTeam(t), address),
			},
			{
				Config: config(""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionNoop)},
				},
			},
		},
	})
}
