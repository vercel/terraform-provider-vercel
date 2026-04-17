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

func TestAcc_ProjectProtectionBypass(t *testing.T) {
	projectSuffix := acctest.RandString(16)
	customSecret := acctest.RandString(32)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             noopDestroyCheck,
		Steps: []resource.TestStep{
			// Single bypass — first one is automatically the env-var default.
			{
				Config: cfg(testAccProjectProtectionBypassSingle(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectProtectionBypassExists(testClient(t), "vercel_project.test", "vercel_project_protection_bypass.first"),
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.first", "is_env_var", "true"),
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.first", "scope", "automation-bypass"),
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.first", "note", "first bypass"),
					resource.TestCheckResourceAttrSet("vercel_project_protection_bypass.first", "secret"),
					resource.TestCheckResourceAttrSet("vercel_project_protection_bypass.first", "created_at"),
				),
			},
			// Add a second bypass with caller-supplied secret. First remains the env-var default.
			{
				Config: cfg(testAccProjectProtectionBypassDouble(projectSuffix, customSecret, "second bypass", false, true)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.first", "is_env_var", "true"),
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.second", "is_env_var", "false"),
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.second", "secret", customSecret),
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.second", "note", "second bypass"),
				),
			},
			// Promote the second bypass to the env-var default. The API atomically demotes the first.
			{
				Config: cfg(testAccProjectProtectionBypassDouble(projectSuffix, customSecret, "second bypass", true, false)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.first", "is_env_var", "false"),
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.second", "is_env_var", "true"),
				),
			},
			// Update the note on the second bypass in place.
			{
				Config: cfg(testAccProjectProtectionBypassDouble(projectSuffix, customSecret, "renamed bypass", true, false)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.second", "note", "renamed bypass"),
					resource.TestCheckResourceAttr("vercel_project_protection_bypass.second", "is_env_var", "true"),
				),
			},
			// Import the second bypass by team_id/project_id/secret.
			{
				ResourceName:      "vercel_project_protection_bypass.second",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["vercel_project_protection_bypass.second"]
					if !ok {
						return "", fmt.Errorf("not found: vercel_project_protection_bypass.second")
					}
					projectID := rs.Primary.Attributes["project_id"]
					teamID := rs.Primary.Attributes["team_id"]
					secret := rs.Primary.Attributes["secret"]
					if teamID == "" {
						return fmt.Sprintf("%s/%s", projectID, secret), nil
					}
					return fmt.Sprintf("%s/%s/%s", teamID, projectID, secret), nil
				},
			},
			// Remove the second bypass — verify it's revoked while the first remains.
			{
				Config: cfg(testAccProjectProtectionBypassSingle(projectSuffix)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectProtectionBypassRevoked(testClient(t), "vercel_project.test", customSecret),
				),
			},
		},
	})
}

func testAccProjectProtectionBypassExists(testClient *client.Client, projectResource, bypassResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		project, ok := s.RootModule().Resources[projectResource]
		if !ok {
			return fmt.Errorf("not found: %s", projectResource)
		}
		bypass, ok := s.RootModule().Resources[bypassResource]
		if !ok {
			return fmt.Errorf("not found: %s", bypassResource)
		}
		secret := bypass.Primary.Attributes["secret"]
		teamID := bypass.Primary.Attributes["team_id"]
		_, err := testClient.GetProtectionBypass(context.TODO(), project.Primary.ID, teamID, secret)
		return err
	}
}

func testAccProjectProtectionBypassRevoked(testClient *client.Client, projectResource, secret string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		project, ok := s.RootModule().Resources[projectResource]
		if !ok {
			return fmt.Errorf("not found: %s", projectResource)
		}
		teamID := project.Primary.Attributes["team_id"]
		_, err := testClient.GetProtectionBypass(context.TODO(), project.Primary.ID, teamID, secret)
		if err == nil {
			return fmt.Errorf("expected bypass %s to be revoked but it still exists", secret)
		}
		if !client.NotFound(err) {
			return fmt.Errorf("unexpected error reading revoked bypass: %s", err)
		}
		return nil
	}
}

func testAccProjectProtectionBypassSingle(projectSuffix string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-bypass-%[1]s"
}

resource "vercel_project_protection_bypass" "first" {
  project_id = vercel_project.test.id
  note       = "first bypass"
}
`, projectSuffix)
}

func testAccProjectProtectionBypassDouble(projectSuffix, customSecret, secondNote string, secondIsEnvVar, firstIsEnvVar bool) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-bypass-%[1]s"
}

resource "vercel_project_protection_bypass" "first" {
  project_id = vercel_project.test.id
  note       = "first bypass"
  is_env_var = %[4]t
}

resource "vercel_project_protection_bypass" "second" {
  project_id = vercel_project.test.id
  note       = "%[3]s"
  secret     = "%[2]s"
  is_env_var = %[5]t
}
`, projectSuffix, customSecret, secondNote, firstIsEnvVar, secondIsEnvVar)
}
