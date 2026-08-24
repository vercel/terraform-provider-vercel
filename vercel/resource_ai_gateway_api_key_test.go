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

func testCheckAIGatewayAPIKeyExists(testClient *client.Client, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetAIGatewayAPIKey(context.TODO(), rs.Primary.ID, rs.Primary.Attributes["team_id"])
		if err != nil {
			return fmt.Errorf("error getting AI Gateway API key %s: %w", rs.Primary.ID, err)
		}
		return nil
	}
}

func testCheckAIGatewayAPIKeyDeleted(testClient *client.Client, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		_, err := testClient.GetAIGatewayAPIKey(context.TODO(), rs.Primary.ID, rs.Primary.Attributes["team_id"])
		if err == nil {
			return fmt.Errorf("expected AI Gateway API key to be deleted")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("unexpected error checking for deleted AI Gateway API key: %w", err)
		}
		return nil
	}
}

func TestAcc_AIGatewayAPIKeyResource(t *testing.T) {
	name := "test-api-key-" + acctest.RandString(16)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckAIGatewayAPIKeyDeleted(testClient(t), "vercel_ai_gateway_api_key.test"),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccAIGatewayAPIKey(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAIGatewayAPIKeyExists(testClient(t), "vercel_ai_gateway_api_key.test"),
					resource.TestCheckResourceAttr("vercel_ai_gateway_api_key.test", "name", name),
					resource.TestCheckResourceAttrSet("vercel_ai_gateway_api_key.test", "id"),
					resource.TestCheckResourceAttrSet("vercel_ai_gateway_api_key.test", "team_id"),
					resource.TestCheckResourceAttrSet("vercel_ai_gateway_api_key.test", "api_key_string"),
					resource.TestCheckResourceAttrSet("vercel_ai_gateway_api_key.test", "partial_key"),
				),
			},
			{
				// Renaming is an in-place update; the key is not replaced and
				// the secret is preserved.
				Config: cfg(testAccAIGatewayAPIKey(name + "-renamed")),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("vercel_ai_gateway_api_key.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAIGatewayAPIKeyExists(testClient(t), "vercel_ai_gateway_api_key.test"),
					resource.TestCheckResourceAttr("vercel_ai_gateway_api_key.test", "name", name+"-renamed"),
					resource.TestCheckResourceAttrSet("vercel_ai_gateway_api_key.test", "api_key_string"),
				),
			},
			{
				ResourceName:            "vercel_ai_gateway_api_key.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"api_key_string"},
			},
		},
	})
}

func TestAcc_AIGatewayAPIKeyResourceWithQuota(t *testing.T) {
	name := "test-api-key-quota-" + acctest.RandString(16)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckAIGatewayAPIKeyDeleted(testClient(t), "vercel_ai_gateway_api_key.test"),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccAIGatewayAPIKeyWithQuota(name, 500, "monthly", "[50, 100]")),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAIGatewayAPIKeyExists(testClient(t), "vercel_ai_gateway_api_key.test"),
					resource.TestCheckResourceAttr("vercel_ai_gateway_api_key.test", "ai_gateway_quota.limit_amount", "500"),
					resource.TestCheckResourceAttr("vercel_ai_gateway_api_key.test", "ai_gateway_quota.refresh_period", "monthly"),
					resource.TestCheckResourceAttr("vercel_ai_gateway_api_key.test", "ai_gateway_quota.alert_thresholds.#", "2"),
				),
			},
			{
				// Quota changes are in-place updates.
				Config: cfg(testAccAIGatewayAPIKeyWithQuota(name, 750, "weekly", "[75]")),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("vercel_ai_gateway_api_key.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_ai_gateway_api_key.test", "ai_gateway_quota.limit_amount", "750"),
					resource.TestCheckResourceAttr("vercel_ai_gateway_api_key.test", "ai_gateway_quota.refresh_period", "weekly"),
					resource.TestCheckResourceAttr("vercel_ai_gateway_api_key.test", "ai_gateway_quota.alert_thresholds.#", "1"),
				),
			},
			{
				// Removing the quota archives it in-place.
				Config: cfg(testAccAIGatewayAPIKey(name)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("vercel_ai_gateway_api_key.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("vercel_ai_gateway_api_key.test", "ai_gateway_quota"),
				),
			},
		},
	})
}

func testAccAIGatewayAPIKey(name string) string {
	return fmt.Sprintf(`
resource "vercel_ai_gateway_api_key" "test" {
  name = "%s"
}
`, name)
}

func testAccAIGatewayAPIKeyWithQuota(name string, limitAmount int, refreshPeriod, alertThresholds string) string {
	return fmt.Sprintf(`
resource "vercel_ai_gateway_api_key" "test" {
  name = "%s"

  ai_gateway_quota = {
    limit_amount     = %d
    refresh_period   = "%s"
    alert_thresholds = %s
  }
}
`, name, limitAmount, refreshPeriod, alertThresholds)
}
