package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAcc_AttackChallengeModeResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccAttackChallengeModeConfigResource(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_attack_challenge_mode.enabled", "enabled", "true"),
					resource.TestCheckResourceAttr("vercel_attack_challenge_mode.disabled", "enabled", "false"),
				),
			},
			{
				ImportState:  true,
				ResourceName: "vercel_attack_challenge_mode.enabled",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["vercel_attack_challenge_mode.enabled"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.ID), nil
				},
			},
			{
				ImportState:  true,
				ResourceName: "vercel_attack_challenge_mode.disabled",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["vercel_attack_challenge_mode.disabled"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.ID), nil
				},
			},
			{
				Config: cfg(testAccAttackChallengeModeConfigResourceUpdated(name)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_attack_challenge_mode.enabled", "enabled", "false"),
				),
			},
		},
	})
}

func testAccAttackChallengeModeConfigResource(name string) string {
	return fmt.Sprintf(`
resource "vercel_project" "enabled" {
    name = "test-acc-%[1]s-enabled"
}

resource "vercel_attack_challenge_mode" "enabled" {
    project_id = vercel_project.enabled.id
    enabled = true
}

resource "vercel_project" "disabled" {
    name = "test-acc-%[1]s-disabled"
}

resource "vercel_attack_challenge_mode" "disabled" {
    project_id = vercel_project.disabled.id
    enabled = false
}
`, name)
}

func testAccAttackChallengeModeConfigResourceUpdated(name string) string {
	return fmt.Sprintf(`
resource "vercel_project" "enabled" {
    name = "test-acc-%[1]s-enabled"
}

resource "vercel_attack_challenge_mode" "enabled" {
    project_id = vercel_project.enabled.id
    enabled = false
}
`, name)
}
