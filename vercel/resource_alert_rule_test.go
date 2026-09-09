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
	notifyOwners := false
	const resourceName = "vercel_alert_rule.example"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckAlertRuleDeleted(testClient(t), resourceName, testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccResourceAlertRule(name, "high", "statusGroup:5xx", nil)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAlertRuleExists(testClient(t), testTeam(t), resourceName),
					resource.TestCheckResourceAttr(resourceName, "type", "built-in"),
					resource.TestCheckResourceAttr(resourceName, "name", "test-acc-alert-rule-"+name),
					resource.TestCheckResourceAttr(resourceName, "rule_scope.type", "include"),
					resource.TestCheckResourceAttr(resourceName, "rule_scope.project_ids.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "triggers.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "triggers.*", map[string]string{"type": "error_anomaly", "filter": "statusGroup:5xx"}),
					resource.TestCheckResourceAttr(resourceName, "match_minimum_severity_level", "high"),
					resource.TestCheckResourceAttr(resourceName, "notification_settings.enable_team_owner_notifications", "true"),
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
				Config: cfg(testAccResourceAlertRule(name, "medium", "statusGroup:5xx", &notifyOwners)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "match_minimum_severity_level", "medium"),
					resource.TestCheckResourceAttr(resourceName, "notification_settings.enable_team_owner_notifications", "false"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "triggers.*", map[string]string{"type": "error_anomaly", "filter": "statusGroup:5xx"}),
				),
			},
		},
	})
}

func TestAcc_AlertRuleResourceCanonicalizedFilter(t *testing.T) {
	name := acctest.RandString(16)
	const resourceName = "vercel_alert_rule.example"
	config := cfg(testAccResourceAlertRule(name, "high", "NOT statusGroup:4xx", nil))
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckAlertRuleDeleted(testClient(t), resourceName, testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAlertRuleExists(testClient(t), testTeam(t), resourceName),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "triggers.*", map[string]string{"type": "error_anomaly", "filter": "NOT statusGroup:4xx"}),
				),
			},
			{
				Config:   config,
				PlanOnly: true,
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

func testAccResourceAlertRule(name, severity, filter string, notifyOwners *bool) string {
	notificationSettings := ""
	if notifyOwners != nil {
		notificationSettings = fmt.Sprintf(`
  notification_settings = {
    enable_team_owner_notifications = %t
  }
`, *notifyOwners)
	}
	return fmt.Sprintf(`
resource "vercel_project" "alert_rule" {
  name = "test-acc-alert-rule-%[1]s"
}

resource "vercel_alert_rule" "example" {
  type = "built-in"
  name = "test-acc-alert-rule-%[1]s"
  rule_scope = {
    type        = "include"
    project_ids = [vercel_project.alert_rule.id]
  }
  triggers = [{
    type   = "error_anomaly"
    filter = "%[3]s"
  }]
  match_minimum_severity_level = "%[2]s"
%[4]s
}
`, name, severity, filter, notificationSettings)
}
