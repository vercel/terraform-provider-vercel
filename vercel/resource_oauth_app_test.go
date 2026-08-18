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

func testCheckOAuthAppExists(testClient *client.Client, teamID, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, err := testClient.GetOAuthApp(context.TODO(), rs.Primary.ID, teamID)
		return err
	}
}

func testCheckOAuthAppDoesNotExist(testClient *client.Client, teamID, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		_, err := testClient.GetOAuthApp(context.TODO(), rs.Primary.ID, teamID)
		if err == nil {
			return fmt.Errorf("expected not_found error, but got no error")
		}
		if !client.OAuthAppNotFound(err) {
			return fmt.Errorf("Unexpected error checking for deleted oauth app: %s", err)
		}

		return nil
	}
}

func getOAuthAppImportID(n string) resource.ImportStateIdFunc {
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

func TestAcc_OAuthAppResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testCheckOAuthAppDoesNotExist(testClient(t), testTeam(t), "vercel_oauth_app.test"),
		),
		Steps: []resource.TestStep{
			{
				// Minimal config: neither redirect_uris nor scopes set, covering
				// the null-set handling in Create.
				Config: cfg(testAccResourceOAuthApp(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckOAuthAppExists(testClient(t), testTeam(t), "vercel_oauth_app.test"),
					resource.TestCheckResourceAttrSet("vercel_oauth_app.test", "id"),
					resource.TestCheckResourceAttr("vercel_oauth_app.test", "name", fmt.Sprintf("test acc %s", name)),
					resource.TestCheckResourceAttr("vercel_oauth_app.test", "slug", fmt.Sprintf("test-acc-%s", name)),
					resource.TestCheckNoResourceAttr("vercel_oauth_app.test", "redirect_uris"),
					// The API force-includes openid even when scopes are unset.
					resource.TestCheckTypeSetElemAttr("vercel_oauth_app.test", "scopes.*", "openid"),
				),
			},
			{
				ResourceName:      "vercel_oauth_app.test",
				ImportState:       true,
				ImportStateIdFunc: getOAuthAppImportID("vercel_oauth_app.test"),
			},
			{
				Config: cfg(testAccResourceOAuthAppUpdated(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckOAuthAppExists(testClient(t), testTeam(t), "vercel_oauth_app.test"),
					resource.TestCheckResourceAttr("vercel_oauth_app.test", "name", fmt.Sprintf("test acc %s updated", name)),
					resource.TestCheckResourceAttr("vercel_oauth_app.test", "description", "An updated description."),
					resource.TestCheckResourceAttr("vercel_oauth_app.test", "home_page_uri", "https://example.com"),
					resource.TestCheckResourceAttr("vercel_oauth_app.test", "redirect_uris.#", "2"),
					resource.TestCheckTypeSetElemAttr("vercel_oauth_app.test", "redirect_uris.*", "https://example.com/other/callback"),
					resource.TestCheckResourceAttr("vercel_oauth_app.test", "scopes.#", "4"),
					resource.TestCheckTypeSetElemAttr("vercel_oauth_app.test", "scopes.*", "offline_access"),
				),
			},
		},
	})
}

func TestAcc_OAuthAppClientSecretResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testCheckOAuthAppDoesNotExist(testClient(t), testTeam(t), "vercel_oauth_app.test"),
		),
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccResourceOAuthAppClientSecret(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckOAuthAppExists(testClient(t), testTeam(t), "vercel_oauth_app.test"),
					resource.TestCheckResourceAttrSet("vercel_oauth_app_client_secret.test", "id"),
					resource.TestCheckResourceAttrSet("vercel_oauth_app_client_secret.test", "client_secret"),
					resource.TestCheckResourceAttrSet("vercel_oauth_app_client_secret.test", "last_four_chars"),
					testCheckOAuthAppClientSecretExists(testClient(t), testTeam(t), "vercel_oauth_app_client_secret.test"),
				),
			},
		},
	})
}

func testCheckOAuthAppClientSecretExists(testClient *client.Client, teamID, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		app, err := testClient.GetOAuthApp(context.TODO(), rs.Primary.Attributes["oauth_app_id"], teamID)
		if err != nil {
			return err
		}
		lastFour := rs.Primary.Attributes["last_four_chars"]
		for _, secret := range app.ClientSecrets {
			if secret.LastFourChars == lastFour {
				return nil
			}
		}
		return fmt.Errorf("no client secret with last four chars %q found on app", lastFour)
	}
}

func testAccResourceOAuthApp(name string) string {
	return fmt.Sprintf(`
resource "vercel_oauth_app" "test" {
	name = "test acc %[1]s"
	slug = "test-acc-%[1]s"
}
`, name)
}

func testAccResourceOAuthAppUpdated(name string) string {
	return fmt.Sprintf(`
resource "vercel_oauth_app" "test" {
	name        = "test acc %[1]s updated"
	slug        = "test-acc-%[1]s"
	description = "An updated description."

	home_page_uri = "https://example.com"

	redirect_uris = [
		"https://example.com/api/auth/callback",
		"https://example.com/other/callback",
	]

	scopes = ["openid", "email", "profile", "offline_access"]
}
`, name)
}

func testAccResourceOAuthAppClientSecret(name string) string {
	return fmt.Sprintf(`
resource "vercel_oauth_app" "test" {
	name = "test acc %[1]s"
	slug = "test-acc-%[1]s"

	redirect_uris = ["https://example.com/api/auth/callback"]
}

resource "vercel_oauth_app_client_secret" "test" {
	oauth_app_id = vercel_oauth_app.test.id
}
`, name)
}
