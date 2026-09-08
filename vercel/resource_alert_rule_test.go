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

func testCheckAlertRuleExists(testClient *client.Client, teamID, name string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		resourceState, ok := state.RootModule().Resources[name]
		if !ok || resourceState.Primary.ID == "" {
			return fmt.Errorf("alert rule not found in state: %s", name)
		}
		_, err := testClient.GetAlertRule(context.Background(), resourceState.Primary.ID, teamID)
		return err
	}
}

func testCheckAlertRuleDeleted(testClient *client.Client, name, teamID string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		resourceState, ok := state.RootModule().Resources[name]
		if !ok || resourceState.Primary.ID == "" {
			return nil
		}
		_, err := testClient.GetAlertRule(context.Background(), resourceState.Primary.ID, teamID)
		if err == nil {
			return fmt.Errorf("alert rule %s still exists", resourceState.Primary.ID)
		}
		if !client.NotFound(err) {
			return err
		}
		return nil
	}
}

func TestAcc_AlertRuleResource(t *testing.T) {
	name := acctest.RandString(16)
	const resourceName = "vercel_alert_rule.example"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckAlertRuleDeleted(testClient(t), resourceName, testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccResourceAlertRule(name, "high", false)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAlertRuleExists(testClient(t), testTeam(t), resourceName),
					resource.TestCheckResourceAttr(resourceName, "type", "built-in"),
					resource.TestCheckResourceAttr(resourceName, "name", "errors-"+name),
					resource.TestCheckResourceAttr(resourceName, "rule_scope.type", "include"),
					resource.TestCheckResourceAttr(resourceName, "rule_scope.project_ids.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "triggers.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "triggers.*", map[string]string{"type": "error_anomaly", "filter": "statusGroup:5xx"}),
					resource.TestCheckResourceAttr(resourceName, "match_minimum_severity_level", "high"),
					resource.TestCheckResourceAttr(resourceName, "notification_settings.enable_team_owner_notifications", "false"),
					resource.TestCheckResourceAttr(resourceName, "is_default", "false"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getAlertRuleImportID(resourceName),
			},
			{
				Config: cfg(testAccResourceAlertRule(name, "medium", true)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "match_minimum_severity_level", "medium"),
					resource.TestCheckResourceAttr(resourceName, "notification_settings.enable_team_owner_notifications", "true"),
				),
			},
		},
	})
}

func getAlertRuleImportID(name string) resource.ImportStateIdFunc {
	return func(state *terraform.State) (string, error) {
		resourceState, ok := state.RootModule().Resources[name]
		if !ok {
			return "", fmt.Errorf("not found: %s", name)
		}
		return fmt.Sprintf("%s/%s", resourceState.Primary.Attributes["team_id"], resourceState.Primary.ID), nil
	}
}

func testAccResourceAlertRule(name, severity string, notifyOwners bool) string {
	return fmt.Sprintf(`
resource "vercel_project" "alert_rule" {
  name = "test-acc-alert-rule-%[1]s"
}

resource "vercel_alert_rule" "example" {
  type = "built-in"
  name = "errors-%[1]s"
  rule_scope = {
    type        = "include"
    project_ids = [vercel_project.alert_rule.id]
  }
  triggers = [{
    type   = "error_anomaly"
    filter = "statusGroup:5xx"
  }]
  match_minimum_severity_level = "%[2]s"
  notification_settings = {
    enable_team_owner_notifications = %[3]t
  }
}
`, name, severity, notifyOwners)
}

func TestAcc_AlertRuleResourceCustom(t *testing.T) {
	if os.Getenv("VERCEL_TERRAFORM_TESTING_OBSERVABILITY_PLUS") == "" {
		t.Skip("VERCEL_TERRAFORM_TESTING_OBSERVABILITY_PLUS is not set")
	}

	name := acctest.RandString(16)
	const resourceName = "vercel_alert_rule.custom"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckAlertRuleDeleted(testClient(t), resourceName, testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccResourceAlertRuleCustom(name, 0.05)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAlertRuleExists(testClient(t), testTeam(t), resourceName),
					resource.TestCheckResourceAttr(resourceName, "type", "custom"),
					resource.TestCheckResourceAttr(resourceName, "rule_scope.type", "project"),
					resource.TestCheckResourceAttr(resourceName, "severity", "medium"),
					resource.TestCheckResourceAttr(resourceName, "evaluation.window", "1h"),
					resource.TestCheckResourceAttr(resourceName, "evaluation.query.metrics.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "evaluation.query.formulas.formula", "errors / requests"),
					resource.TestCheckResourceAttr(resourceName, "trigger.type", "threshold"),
					resource.TestCheckResourceAttr(resourceName, "trigger.threshold", "0.05"),
					resource.TestCheckResourceAttr(resourceName, "query_supported", "true"),
				),
			},
			{
				Config: cfg(testAccResourceAlertRuleCustom(name, 0.1)),
				Check:  resource.TestCheckResourceAttr(resourceName, "trigger.threshold", "0.1"),
			},
		},
	})
}

func testAccResourceAlertRuleCustom(name string, threshold float64) string {
	return fmt.Sprintf(`
resource "vercel_project" "custom_alert_rule" {
  name = "test-acc-custom-alert-rule-%[1]s"
}

resource "vercel_alert_rule" "custom" {
  type = "custom"
  name = "error-rate-%[1]s"
  rule_scope = {
    type       = "project"
    project_id = vercel_project.custom_alert_rule.id
  }
  severity = "medium"
  evaluation = {
    window = "1h"
    query = {
      metrics = {
        errors = {
          metric      = "vercel.request.count"
          aggregation = "sum"
          filter      = "httpStatus>=500"
        }
        requests = {
          metric      = "vercel.request.count"
          aggregation = "sum"
        }
      }
      formulas = { formula = "errors / requests" }
      outputs  = ["formula"]
    }
  }
  trigger = {
    type      = "threshold"
    output    = "formula"
    operator  = "gt"
    threshold = %[2]g
    minimum = {
      output    = "errors"
      threshold = 20
    }
  }
}
`, name, threshold)
}
