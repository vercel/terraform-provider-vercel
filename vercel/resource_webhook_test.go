package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vercel/terraform-provider-vercel/v2/client"
)

func testCheckWebhookExists(teamID, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient().GetWebhook(context.TODO(), rs.Primary.ID, teamID)
		return err
	}
}

func testCheckWebhooksDeleted(n1, n2, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, n := range []string{n1, n2} {
			rs, ok := s.RootModule().Resources[n]
			if !ok {
				return fmt.Errorf("not found: %s", n)
			}

			if rs.Primary.ID == "" {
				return fmt.Errorf("no ID is set")
			}

			_, err := testClient().GetWebhook(context.TODO(), rs.Primary.ID, teamID)
			if err == nil {
				return fmt.Errorf("expected not_found error, but got no error")
			}
			if !client.NotFound(err) {
				return fmt.Errorf("Unexpected error checking for deleted webhook: %s", err)
			}
		}
		return nil
	}
}

func TestAcc_WebhookResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckWebhooksDeleted("vercel_webhook.with_project_ids", "vercel_webhook.without_project_ids", testTeam()),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceWebhook(name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckWebhookExists(testTeam(), "vercel_webhook.with_project_ids"),
					resource.TestCheckTypeSetElemAttr("vercel_webhook.with_project_ids", "events.*", "deployment.created"),
					resource.TestCheckTypeSetElemAttr("vercel_webhook.with_project_ids", "events.*", "deployment.succeeded"),
					resource.TestCheckResourceAttrSet("vercel_webhook.with_project_ids", "id"),
					resource.TestCheckResourceAttrSet("vercel_webhook.with_project_ids", "secret"),

					testCheckWebhookExists(testTeam(), "vercel_webhook.without_project_ids"),
					resource.TestCheckTypeSetElemAttr("vercel_webhook.without_project_ids", "events.*", "deployment.created"),
					resource.TestCheckTypeSetElemAttr("vercel_webhook.without_project_ids", "events.*", "deployment.succeeded"),
					resource.TestCheckResourceAttrSet("vercel_webhook.without_project_ids", "id"),
					resource.TestCheckResourceAttrSet("vercel_webhook.without_project_ids", "secret"),
				),
			},
		},
	})
}

func testAccResourceWebhook(name, team string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
    name = "test-acc-%[1]s"
    %[2]s
}

resource "vercel_project" "test2" {
    name = "test-acc-%[1]s-2"
    %[2]s
}

resource "vercel_webhook" "with_project_ids" {
    events = ["deployment.created", "deployment.succeeded"]
    endpoint = "https://example.com/foo"
    project_ids = [vercel_project.test.id, vercel_project.test2.id]
    %[2]s
}

resource "vercel_webhook" "without_project_ids" {
    events = ["deployment.created", "deployment.succeeded"]
    endpoint = "https://example.com/foo"
    project_ids = [vercel_project.test.id, vercel_project.test2.id]
    %[2]s
}
`, name, team)
}
