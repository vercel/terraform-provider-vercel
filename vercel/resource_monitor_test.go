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

func testCheckMonitorExists(testClient *client.Client, teamID, name string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		resourceState, ok := state.RootModule().Resources[name]
		if !ok || resourceState.Primary.ID == "" {
			return fmt.Errorf("monitor not found in state: %s", name)
		}
		monitor, err := testClient.GetMonitor(context.Background(), resourceState.Primary.ID, teamID)
		if err != nil {
			return err
		}
		for _, tag := range monitor.Tags {
			if tag == "terraform-provider-vercel" {
				return nil
			}
		}
		return fmt.Errorf("monitor %s is missing the terraform-provider-vercel tag, tags = %v", monitor.ID, monitor.Tags)
	}
}

func testCheckMonitorDeleted(testClient *client.Client, name, teamID string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		resourceState, ok := state.RootModule().Resources[name]
		if !ok || resourceState.Primary.ID == "" {
			return nil
		}
		_, err := testClient.GetMonitor(context.Background(), resourceState.Primary.ID, teamID)
		if err == nil {
			return fmt.Errorf("monitor %s still exists", resourceState.Primary.ID)
		}
		if !client.NotFound(err) {
			return err
		}
		return nil
	}
}

func TestAcc_MonitorResource(t *testing.T) {
	name := acctest.RandString(16)
	notifyOwners := false
	const resourceName = "vercel_monitor.example"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckMonitorDeleted(testClient(t), resourceName, testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccResourceMonitor(name, "high", "statusGroup:5xx", nil)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckMonitorExists(testClient(t), testTeam(t), resourceName),
					resource.TestCheckResourceAttr(resourceName, "type", "built-in"),
					resource.TestCheckResourceAttr(resourceName, "name", "test-acc-monitor-"+name),
					resource.TestCheckResourceAttr(resourceName, "rule_scope.type", "include"),
					resource.TestCheckResourceAttr(resourceName, "rule_scope.project_ids.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "triggers.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "triggers.*", map[string]string{"type": "error_anomaly", "filter": "statusGroup:5xx"}),
					resource.TestCheckResourceAttr(resourceName, "match_minimum_severity_level", "high"),
					resource.TestCheckTypeSetElemAttr(resourceName, "tags.*", "test-acc"),
					resource.TestCheckResourceAttr(resourceName, "notification_settings.enable_team_owner_notifications", "true"),
					resource.TestCheckResourceAttr(resourceName, "is_default", "false"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getMonitorImportID(resourceName),
			},
			{
				Config: cfg(testAccResourceMonitor(name, "medium", "statusGroup:5xx", &notifyOwners)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "match_minimum_severity_level", "medium"),
					resource.TestCheckResourceAttr(resourceName, "notification_settings.enable_team_owner_notifications", "false"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "triggers.*", map[string]string{"type": "error_anomaly", "filter": "statusGroup:5xx"}),
				),
			},
		},
	})
}

func getMonitorImportID(name string) resource.ImportStateIdFunc {
	return func(state *terraform.State) (string, error) {
		resourceState, ok := state.RootModule().Resources[name]
		if !ok {
			return "", fmt.Errorf("not found: %s", name)
		}
		return fmt.Sprintf("%s/%s", resourceState.Primary.Attributes["team_id"], resourceState.Primary.ID), nil
	}
}

func testAccResourceMonitor(name, severity, filter string, notifyOwners *bool) string {
	notificationSettings := ""
	if notifyOwners != nil {
		notificationSettings = fmt.Sprintf(`
  notification_settings = {
    enable_team_owner_notifications = %t
  }
`, *notifyOwners)
	}
	return fmt.Sprintf(`
resource "vercel_project" "monitor" {
  name = "test-acc-monitor-%[1]s"
}

resource "vercel_monitor" "example" {
  type = "built-in"
  name = "test-acc-monitor-%[1]s"
  rule_scope = {
    type        = "include"
    project_ids = [vercel_project.monitor.id]
  }
  triggers = [{
    type   = "error_anomaly"
    filter = "%[3]s"
  }]
  match_minimum_severity_level = "%[2]s"
  tags                         = ["test-acc"]
%[4]s
}
`, name, severity, filter, notificationSettings)
}
