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

func testCheckAuditLogDrainExists(testClient *client.Client, teamID, name string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		resourceState, ok := state.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}
		if resourceState.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetAuditLogDrain(context.TODO(), resourceState.Primary.ID, teamID)
		return err
	}
}

func testCheckAuditLogDrainDeleted(testClient *client.Client, name, teamID string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		resourceState, ok := state.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}
		if resourceState.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetAuditLogDrain(context.TODO(), resourceState.Primary.ID, teamID)
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("unexpected error checking for deleted Audit Log Drain: %s", err)
		}
		return nil
	}
}

func TestAcc_AuditLogDrainResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckAuditLogDrainDeleted(testClient(t), "vercel_audit_log_drain.test", testTeam(t)),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccResourceAuditLogDrain(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAuditLogDrainExists(testClient(t), testTeam(t), "vercel_audit_log_drain.test"),
					resource.TestCheckResourceAttr("vercel_audit_log_drain.test", "name", "audit-"+name),
					resource.TestCheckResourceAttr("vercel_audit_log_drain.test", "http.0.endpoint", "https://verify-test-rouge.vercel.app/api"),
					resource.TestCheckResourceAttr("vercel_audit_log_drain.test", "http.0.encoding", "json"),
					resource.TestCheckResourceAttr("vercel_audit_log_drain.test", "http.0.compression", "gzip"),
					resource.TestCheckResourceAttr("vercel_audit_log_drain.test", "http.0.headers.%", "1"),
					resource.TestCheckResourceAttrSet("vercel_audit_log_drain.test", "id"),
					resource.TestCheckResourceAttrSet("vercel_audit_log_drain.test", "team_id"),
				),
			},
		},
	})
}

func testAccResourceAuditLogDrain(name string) string {
	return fmt.Sprintf(`
resource "vercel_audit_log_drain" "test" {
    name = "audit-%[1]s"

    http = {
        endpoint    = "https://verify-test-rouge.vercel.app/api"
        encoding    = "json"
        compression = "gzip"
        headers = {
            some-key = "some-value"
        }
        secret = "a_very_long_and_very_well_specified_secret"
    }
}
`, name)
}
