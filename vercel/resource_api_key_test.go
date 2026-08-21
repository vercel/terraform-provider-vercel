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

func testCheckAPIKeyExists(testClient *client.Client, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetAPIKey(context.TODO(), rs.Primary.ID, rs.Primary.Attributes["team_id"])
		if err != nil {
			return fmt.Errorf("error getting API key %s: %w", rs.Primary.ID, err)
		}
		return nil
	}
}

func testCheckAPIKeyDeleted(testClient *client.Client, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		_, err := testClient.GetAPIKey(context.TODO(), rs.Primary.ID, rs.Primary.Attributes["team_id"])
		if err == nil {
			return fmt.Errorf("expected API key to be deleted")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("unexpected error checking for deleted API key: %w", err)
		}
		return nil
	}
}

func TestAcc_APIKeyResource(t *testing.T) {
	name := "test-api-key-" + acctest.RandString(16)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckAPIKeyDeleted(testClient(t), "vercel_api_key.test"),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccAPIKey(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAPIKeyExists(testClient(t), "vercel_api_key.test"),
					resource.TestCheckResourceAttr("vercel_api_key.test", "name", name),
					resource.TestCheckResourceAttr("vercel_api_key.test", "purpose", "ai-gateway"),
					resource.TestCheckResourceAttrSet("vercel_api_key.test", "id"),
					resource.TestCheckResourceAttrSet("vercel_api_key.test", "team_id"),
					resource.TestCheckResourceAttrSet("vercel_api_key.test", "api_key_string"),
					resource.TestCheckResourceAttrSet("vercel_api_key.test", "partial_key"),
					resource.TestCheckResourceAttrSet("vercel_api_key.test", "created_at"),
				),
			},
			{
				ResourceName:            "vercel_api_key.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"api_key_string"},
			},
		},
	})
}

func testAccAPIKey(name string) string {
	return fmt.Sprintf(`
resource "vercel_api_key" "test" {
  name    = "%s"
  purpose = "ai-gateway"
}
`, name)
}
