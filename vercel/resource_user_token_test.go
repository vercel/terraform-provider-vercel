package vercel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vercel/terraform-provider-vercel/v4/client"
)

func testCheckUserTokenExists(testClient *client.Client, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetUserToken(context.TODO(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting token %s: %w", rs.Primary.ID, err)
		}
		return nil
	}
}

func testCheckUserTokenDeleted(testClient *client.Client, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetUserToken(context.TODO(), rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.NotFound(err) {
			return fmt.Errorf("unexpected error checking for deleted user token: %s", err)
		}

		return nil
	}
}

func TestAcc_UserTokenResource(t *testing.T) {
	name := "test-token-" + acctest.RandString(16)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckUserTokenDeleted(testClient(t), "vercel_user_token.test"),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccUserToken(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckUserTokenExists(testClient(t), "vercel_user_token.test"),
					resource.TestCheckResourceAttr("vercel_user_token.test", "name", name),
					resource.TestCheckResourceAttrSet("vercel_user_token.test", "id"),
					resource.TestCheckResourceAttrSet("vercel_user_token.test", "team_id"),
					resource.TestCheckResourceAttrSet("vercel_user_token.test", "bearer_token"),
					resource.TestCheckResourceAttr("vercel_user_token.test", "type", "token"),
					resource.TestCheckResourceAttr("vercel_user_token.test", "origin", "manual"),
					resource.TestCheckResourceAttr("vercel_user_token.test", "prefix", "vcp_"),
					resource.TestCheckResourceAttrSet("vercel_user_token.test", "suffix"),
					resource.TestCheckResourceAttrSet("vercel_user_token.test", "created_at"),
					resource.TestCheckResourceAttrSet("vercel_user_token.test", "active_at"),
				),
			},
			{
				ResourceName:            "vercel_user_token.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"bearer_token"},
			},
		},
	})
}

func testAccUserToken(name string) string {
	return fmt.Sprintf(`
resource "vercel_user_token" "test" {
  name = "%s"
}
`, name)
}
