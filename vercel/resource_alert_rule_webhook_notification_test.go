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

func testCheckAlertRuleWebhookNotificationExists(testClient *client.Client, teamID, name string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		resourceState, ok := state.RootModule().Resources[name]
		if !ok || resourceState.Primary.ID == "" {
			return fmt.Errorf("alert rule notification not found in state: %s", name)
		}
		notifications, err := testClient.GetAlertRuleNotifications(
			context.Background(),
			resourceState.Primary.Attributes["alert_rule_id"],
			teamID,
		)
		if err != nil {
			return err
		}
		webhookID := resourceState.Primary.Attributes["webhook_id"]
		for _, notification := range notifications {
			if notification.Type == client.AlertRuleNotificationTypeWebhook && notification.Webhook.ID == webhookID {
				return nil
			}
		}
		return fmt.Errorf("webhook %s is not linked to alert rule", webhookID)
	}
}

func testCheckAlertRuleWebhookNotificationDeleted(testClient *client.Client, teamID, name string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		resourceState, ok := state.RootModule().Resources[name]
		if !ok || resourceState.Primary.ID == "" {
			return nil
		}
		notifications, err := testClient.GetAlertRuleNotifications(
			context.Background(),
			resourceState.Primary.Attributes["alert_rule_id"],
			teamID,
		)
		if client.NotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		webhookID := resourceState.Primary.Attributes["webhook_id"]
		for _, notification := range notifications {
			if notification.Type == client.AlertRuleNotificationTypeWebhook && notification.Webhook.ID == webhookID {
				return fmt.Errorf("webhook %s is still linked to alert rule", webhookID)
			}
		}
		return nil
	}
}

func TestAcc_AlertRuleWebhookNotificationResource(t *testing.T) {
	name := acctest.RandString(16)
	const resourceName = "vercel_alert_rule_webhook_notification.webhook"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckAlertRuleWebhookNotificationDeleted(testClient(t), testTeam(t), resourceName),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccResourceAlertRuleWebhookNotification(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAlertRuleWebhookNotificationExists(testClient(t), testTeam(t), resourceName),
					resource.TestCheckResourceAttrPair(resourceName, "alert_rule_id", "vercel_alert_rule.notification", "id"),
					resource.TestCheckResourceAttrPair(resourceName, "webhook_id", "vercel_webhook.notification", "id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getAlertRuleNotificationImportID(resourceName),
			},
		},
	})
}

func getAlertRuleNotificationImportID(name string) resource.ImportStateIdFunc {
	return func(state *terraform.State) (string, error) {
		resourceState, ok := state.RootModule().Resources[name]
		if !ok || resourceState.Primary.ID == "" {
			return "", fmt.Errorf("alert rule notification not found in state: %s", name)
		}
		return fmt.Sprintf("%s/%s", resourceState.Primary.Attributes["team_id"], resourceState.Primary.ID), nil
	}
}

func testAccResourceAlertRuleWebhookNotification(name string) string {
	return fmt.Sprintf(`
resource "vercel_project" "alert_rule_notification" {
  name = "test-acc-alert-rule-notification-%[1]s"
}

resource "vercel_alert_rule" "notification" {
  type = "built-in"
  name = "test-acc-alert-rule-notification-%[1]s"
  rule_scope = {
    type        = "include"
    project_ids = [vercel_project.alert_rule_notification.id]
  }
  triggers = [{
    type   = "error_anomaly"
    filter = "statusGroup:5xx"
  }]
  match_minimum_severity_level = "high"
}

resource "vercel_webhook" "notification" {
  endpoint = "https://example.com/vercel-alerts/%[1]s"
  events   = ["alerts.triggered"]
}

resource "vercel_alert_rule_webhook_notification" "webhook" {
  alert_rule_id = vercel_alert_rule.notification.id
  webhook_id    = vercel_webhook.notification.id
}
`, name)
}
