package vercel_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func testCheckAlertRuleExists(testClient *client.Client, teamID, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetAlertRule(context.TODO(), rs.Primary.ID, teamID)
		return err
	}
}

func testCheckAlertRuleDeleted(testClient *client.Client, n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetAlertRule(context.TODO(), rs.Primary.ID, teamID)
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("unexpected error checking for deleted alert rule: %s", err)
		}

		return nil
	}
}

func TestAcc_AlertRuleResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckAlertRuleDeleted(testClient(t), "vercel_alert_rule.team_wide", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccResourceAlertRule(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAlertRuleExists(testClient(t), testTeam(t), "vercel_alert_rule.team_wide"),
					resource.TestCheckResourceAttr("vercel_alert_rule.team_wide", "name", "team-wide-"+name),
					resource.TestCheckResourceAttr("vercel_alert_rule.team_wide", "alert_types.#", "1"),
					resource.TestCheckResourceAttr("vercel_alert_rule.team_wide", "alert_types.0.type", "usage_anomaly"),
					resource.TestCheckNoResourceAttr("vercel_alert_rule.team_wide", "project_filter"),
					resource.TestCheckResourceAttrSet("vercel_alert_rule.team_wide", "id"),
					resource.TestCheckResourceAttrSet("vercel_alert_rule.team_wide", "team_id"),

					testCheckAlertRuleExists(testClient(t), testTeam(t), "vercel_alert_rule.scoped"),
					resource.TestCheckResourceAttr("vercel_alert_rule.scoped", "alert_types.#", "2"),
					resource.TestCheckResourceAttr("vercel_alert_rule.scoped", "alert_types.0.type", "error_anomaly"),
					resource.TestCheckResourceAttr("vercel_alert_rule.scoped", "alert_types.0.filter", "statusGroup eq '5xx'"),
					resource.TestCheckResourceAttr("vercel_alert_rule.scoped", "alert_types.1.type", "usage_anomaly"),
					resource.TestCheckResourceAttr("vercel_alert_rule.scoped", "sensitivity_level", "3"),
					resource.TestCheckResourceAttr("vercel_alert_rule.scoped", "autosubscribe_owners", "false"),
					resource.TestCheckResourceAttr("vercel_alert_rule.scoped", "autosubscribe_project_admins", "false"),
					resource.TestCheckResourceAttrSet("vercel_alert_rule.scoped", "project_filter"),

					resource.TestCheckResourceAttrPair(
						"data.vercel_alert_rule.scoped", "name",
						"vercel_alert_rule.scoped", "name",
					),
					resource.TestCheckResourceAttr("data.vercel_alert_rule.scoped", "is_default", "false"),
					resource.TestCheckResourceAttr("data.vercel_alert_rule.scoped", "sensitivity_level", "3"),
				),
			},
			{
				ResourceName:      "vercel_alert_rule.scoped",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getAlertRuleImportID("vercel_alert_rule.scoped"),
			},
			{
				// Renaming, retargeting and clearing optional fields all happen
				// in place.
				Config: cfg(testAccResourceAlertRuleUpdated(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAlertRuleExists(testClient(t), testTeam(t), "vercel_alert_rule.scoped"),
					resource.TestCheckResourceAttr("vercel_alert_rule.scoped", "name", "scoped-updated-"+name),
					resource.TestCheckResourceAttr("vercel_alert_rule.scoped", "alert_types.#", "1"),
					resource.TestCheckResourceAttr("vercel_alert_rule.scoped", "alert_types.0.type", "error_anomaly"),
					resource.TestCheckNoResourceAttr("vercel_alert_rule.scoped", "alert_types.0.filter"),
					resource.TestCheckNoResourceAttr("vercel_alert_rule.scoped", "sensitivity_level"),
					resource.TestCheckNoResourceAttr("vercel_alert_rule.scoped", "project_filter"),
				),
			},
		},
	})
}

func getAlertRuleImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}

		return fmt.Sprintf("%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.ID), nil
	}
}

func testAccResourceAlertRule(name string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
    name = "test-acc-%[1]s"
}

resource "vercel_alert_rule" "team_wide" {
    name = "team-wide-%[1]s"
    alert_types = [{
        type = "usage_anomaly"
    }]
    autosubscribe_owners         = false
    autosubscribe_project_admins = false
}

resource "vercel_alert_rule" "scoped" {
    name           = "scoped-%[1]s"
    project_filter = "projectId in ('${vercel_project.test.id}')"
    alert_types = [
        {
            type   = "error_anomaly"
            filter = "statusGroup eq '5xx'"
        },
        {
            type = "usage_anomaly"
        },
    ]
    sensitivity_level            = 3
    autosubscribe_owners         = false
    autosubscribe_project_admins = false
}

data "vercel_alert_rule" "scoped" {
    id = vercel_alert_rule.scoped.id
}
`, name)
}

func testAccResourceAlertRuleUpdated(name string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
    name = "test-acc-%[1]s"
}

resource "vercel_alert_rule" "team_wide" {
    name = "team-wide-%[1]s"
    alert_types = [{
        type = "usage_anomaly"
    }]
    autosubscribe_owners         = false
    autosubscribe_project_admins = false
}

resource "vercel_alert_rule" "scoped" {
    name = "scoped-updated-%[1]s"
    alert_types = [{
        type = "error_anomaly"
    }]
    autosubscribe_owners         = false
    autosubscribe_project_admins = false
}
`, name)
}

// TestAcc_AlertRuleResourceCustomAlert covers custom Observability metric
// alerts. The API rejects these unless the team has an Observability Plus
// subscription with a non-zero custom alert allowance, which the shared testing
// team does not, so this test is opt-in.
func TestAcc_AlertRuleResourceCustomAlert(t *testing.T) {
	if os.Getenv("VERCEL_TERRAFORM_TESTING_OBSERVABILITY_PLUS") == "" {
		t.Skip("skipping: custom alert rules require VERCEL_TERRAFORM_TESTING_OBSERVABILITY_PLUS to be set, for a team with an Observability Plus subscription")
	}

	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckAlertRuleDeleted(testClient(t), "vercel_alert_rule.custom", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccResourceAlertRuleCustomAlert(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAlertRuleExists(testClient(t), testTeam(t), "vercel_alert_rule.custom"),
					resource.TestCheckResourceAttr("vercel_alert_rule.custom", "alert_types.0.type", "custom_alert"),
					resource.TestCheckResourceAttrSet("vercel_alert_rule.custom", "project_id"),
					resource.TestCheckNoResourceAttr("vercel_alert_rule.custom", "project_filter"),
					resource.TestCheckResourceAttr("vercel_alert_rule.custom", "custom_alert.event", "incomingRequest"),
					resource.TestCheckResourceAttr("vercel_alert_rule.custom", "custom_alert.rollups.%", "2"),
					resource.TestCheckResourceAttr("vercel_alert_rule.custom", "custom_alert.rollups.errors.filter", "httpStatus ge 500"),
					resource.TestCheckResourceAttr("vercel_alert_rule.custom", "custom_alert.granularity", "1h"),
					resource.TestCheckResourceAttr("vercel_alert_rule.custom", "custom_alert.trigger_type", "threshold"),
					resource.TestCheckResourceAttr("vercel_alert_rule.custom", "custom_alert.trigger_operator", "gt"),
					resource.TestCheckResourceAttr("vercel_alert_rule.custom", "custom_alert.trigger_threshold", "0.05"),
					resource.TestCheckResourceAttr("vercel_alert_rule.custom", "custom_alert.min_threshold", "20"),
					resource.TestCheckResourceAttr("vercel_alert_rule.custom", "custom_alert.formula.operator", "divide"),
					resource.TestCheckResourceAttr("vercel_alert_rule.custom", "custom_alert.formula.left", "errors"),
					resource.TestCheckResourceAttr("vercel_alert_rule.custom", "custom_alert.formula.right", "requests"),
				),
			},
			{
				Config: cfg(testAccResourceAlertRuleCustomAlertUpdated(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAlertRuleExists(testClient(t), testTeam(t), "vercel_alert_rule.custom"),
					resource.TestCheckResourceAttr("vercel_alert_rule.custom", "custom_alert.trigger_threshold", "0.1"),
					resource.TestCheckResourceAttr("vercel_alert_rule.custom", "custom_alert.granularity", "5m"),
				),
			},
		},
	})
}

func testAccResourceAlertRuleCustomAlert(name string) string {
	return fmt.Sprintf(`
resource "vercel_project" "custom" {
    name = "test-acc-custom-%[1]s"
}

resource "vercel_alert_rule" "custom" {
    name       = "custom-%[1]s"
    project_id = vercel_project.custom.id
    alert_types = [{
        type = "custom_alert"
    }]
    autosubscribe_owners         = false
    autosubscribe_project_admins = false

    custom_alert = {
        event = "incomingRequest"
        rollups = {
            errors = {
                measure     = "count"
                aggregation = "sum"
                filter      = "httpStatus ge 500"
            }
            requests = {
                measure     = "count"
                aggregation = "sum"
            }
        }
        granularity       = "1h"
        trigger_type      = "threshold"
        trigger_operator  = "gt"
        trigger_threshold = 0.05
        min_threshold     = 20
        formula = {
            operator = "divide"
            left     = "errors"
            right    = "requests"
        }
    }
}
`, name)
}

func testAccResourceAlertRuleCustomAlertUpdated(name string) string {
	return fmt.Sprintf(`
resource "vercel_project" "custom" {
    name = "test-acc-custom-%[1]s"
}

resource "vercel_alert_rule" "custom" {
    name       = "custom-%[1]s"
    project_id = vercel_project.custom.id
    alert_types = [{
        type = "custom_alert"
    }]
    autosubscribe_owners         = false
    autosubscribe_project_admins = false

    custom_alert = {
        event = "incomingRequest"
        rollups = {
            errors = {
                measure     = "count"
                aggregation = "sum"
                filter      = "httpStatus ge 500"
            }
            requests = {
                measure     = "count"
                aggregation = "sum"
            }
        }
        granularity       = "5m"
        trigger_type      = "threshold"
        trigger_operator  = "gt"
        trigger_threshold = 0.1
        min_threshold     = 20
        formula = {
            operator = "divide"
            left     = "errors"
            right    = "requests"
        }
    }
}
`, name)
}
