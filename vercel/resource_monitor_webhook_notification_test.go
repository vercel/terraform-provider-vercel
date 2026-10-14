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

func testCheckMonitorWebhookNotificationExists(testClient *client.Client, teamID, name string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		resourceState, ok := state.RootModule().Resources[name]
		if !ok || resourceState.Primary.ID == "" {
			return fmt.Errorf("monitor notification not found in state: %s", name)
		}
		notifications, err := testClient.GetMonitorNotifications(
			context.Background(),
			resourceState.Primary.Attributes["monitor_id"],
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
		return fmt.Errorf("webhook %s is not linked to monitor", webhookID)
	}
}

func testCheckMonitorWebhookNotificationDeleted(testClient *client.Client, teamID, name string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		resourceState, ok := state.RootModule().Resources[name]
		if !ok || resourceState.Primary.ID == "" {
			return nil
		}
		notifications, err := testClient.GetMonitorNotifications(
			context.Background(),
			resourceState.Primary.Attributes["monitor_id"],
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
				return fmt.Errorf("webhook %s is still linked to monitor", webhookID)
			}
		}
		return nil
	}
}

func TestAcc_MonitorWebhookNotificationResource(t *testing.T) {
	name := acctest.RandString(16)
	const resourceName = "vercel_monitor_webhook_notification.webhook"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckMonitorWebhookNotificationDeleted(testClient(t), testTeam(t), resourceName),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccResourceMonitorWebhookNotification(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckMonitorWebhookNotificationExists(testClient(t), testTeam(t), resourceName),
					resource.TestCheckResourceAttrPair(resourceName, "monitor_id", "vercel_monitor.notification", "id"),
					resource.TestCheckResourceAttrPair(resourceName, "webhook_id", "vercel_webhook.notification", "id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getMonitorNotificationImportID(resourceName),
			},
		},
	})
}

func getMonitorNotificationImportID(name string) resource.ImportStateIdFunc {
	return func(state *terraform.State) (string, error) {
		resourceState, ok := state.RootModule().Resources[name]
		if !ok || resourceState.Primary.ID == "" {
			return "", fmt.Errorf("monitor notification not found in state: %s", name)
		}
		return fmt.Sprintf("%s/%s", resourceState.Primary.Attributes["team_id"], resourceState.Primary.ID), nil
	}
}

func testAccResourceMonitorWebhookNotification(name string) string {
	return fmt.Sprintf(`
resource "vercel_project" "monitor_notification" {
  name = "test-acc-monitor-notification-%[1]s"
}

resource "vercel_monitor" "notification" {
  type = "built-in"
  name = "test-acc-monitor-notification-%[1]s"
  rule_scope = {
    type        = "include"
    project_ids = [vercel_project.monitor_notification.id]
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

resource "vercel_monitor_webhook_notification" "webhook" {
  monitor_id = vercel_monitor.notification.id
  webhook_id    = vercel_webhook.notification.id
}
`, name)
}
