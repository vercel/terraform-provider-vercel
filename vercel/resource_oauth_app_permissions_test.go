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

func testCheckOAuthAppPermissionsMatch(testClient *client.Client, teamID, n string, want []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		app, err := testClient.GetOAuthApp(context.TODO(), rs.Primary.Attributes["oauth_app_id"], teamID)
		if err != nil {
			return err
		}
		granted := map[string]bool{}
		for _, permission := range app.Permissions {
			granted[permission] = true
		}
		if len(app.Permissions) != len(want) {
			return fmt.Errorf("expected %d permissions, API reports %v", len(want), app.Permissions)
		}
		for _, permission := range want {
			if !granted[permission] {
				return fmt.Errorf("expected permission %q to be granted, API reports %v", permission, app.Permissions)
			}
		}
		return nil
	}
}

// The app outlives the grant in the final step, so revocation (not just app
// deletion) is what's being verified.
func testCheckOAuthAppPermissionsRevoked(testClient *client.Client, teamID, appResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[appResource]
		if !ok {
			return fmt.Errorf("not found: %s", appResource)
		}

		app, err := testClient.GetOAuthApp(context.TODO(), rs.Primary.ID, teamID)
		if err != nil {
			return err
		}
		if len(app.Permissions) != 0 {
			return fmt.Errorf("expected all permissions revoked, API reports %v", app.Permissions)
		}
		return nil
	}
}

func TestAcc_OAuthAppPermissionsResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testCheckOAuthAppDoesNotExist(testClient(t), testTeam(t), "vercel_oauth_app.test"),
		),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccResourceOAuthAppPermissions(name, `["read:team", "read:deployment"]`)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckOAuthAppExists(testClient(t), testTeam(t), "vercel_oauth_app.test"),
					resource.TestCheckResourceAttrSet("vercel_oauth_app_permissions.test", "id"),
					resource.TestCheckResourceAttr("vercel_oauth_app_permissions.test", "permissions.#", "2"),
					resource.TestCheckTypeSetElemAttr("vercel_oauth_app_permissions.test", "permissions.*", "read:team"),
					testCheckOAuthAppPermissionsMatch(testClient(t), testTeam(t), "vercel_oauth_app_permissions.test",
						[]string{"read:team", "read:deployment"}),
				),
			},
			{
				ResourceName:      "vercel_oauth_app_permissions.test",
				ImportState:       true,
				ImportStateIdFunc: getOAuthAppPermissionsImportID("vercel_oauth_app_permissions.test"),
			},
			{
				Config: cfg(testAccResourceOAuthAppPermissions(name, `["read:team", "read-write:deployment", "read-write:project"]`)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_oauth_app_permissions.test", "permissions.#", "3"),
					testCheckOAuthAppPermissionsMatch(testClient(t), testTeam(t), "vercel_oauth_app_permissions.test",
						[]string{"read:team", "read-write:deployment", "read-write:project"}),
				),
			},
			{
				// Remove the grant resource while KEEPING the app: destroying the
				// grant must revoke every permission on the still-existing app.
				Config: cfg(testAccResourceOAuthAppPermissionsAppOnly(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckOAuthAppExists(testClient(t), testTeam(t), "vercel_oauth_app.test"),
					testCheckOAuthAppPermissionsRevoked(testClient(t), testTeam(t), "vercel_oauth_app.test"),
				),
			},
		},
	})
}

func getOAuthAppPermissionsImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return "", fmt.Errorf("no ID is set")
		}
		return fmt.Sprintf("%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.ID), nil
	}
}

func testAccResourceOAuthAppPermissionsAppOnly(name string) string {
	return fmt.Sprintf(`
resource "vercel_oauth_app" "test" {
	name = "test acc %[1]s"
	slug = "test-acc-%[1]s"
}
`, name)
}

func testAccResourceOAuthAppPermissions(name, permissions string) string {
	return fmt.Sprintf(`
resource "vercel_oauth_app" "test" {
	name = "test acc %[1]s"
	slug = "test-acc-%[1]s"
}

resource "vercel_oauth_app_permissions" "test" {
	oauth_app_id = vercel_oauth_app.test.id

	permissions = %[2]s
}
`, name, permissions)
}
