package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_AttackChallengeModeDataSource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAttackChallengeModeConfig(name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_attack_challenge_mode.never_enabled", "enabled", "false"),
					resource.TestCheckResourceAttr("data.vercel_attack_challenge_mode.enabled", "enabled", "true"),
					resource.TestCheckResourceAttr("data.vercel_attack_challenge_mode.disabled", "enabled", "false"),
				),
			},
		},
	})
}

func testAccAttackChallengeModeConfig(name, teamID string) string {
	return fmt.Sprintf(`
resource "vercel_project" "never_enabled" {
    name = "test-acc-%[1]s"
    %[2]s
}

data "vercel_attack_challenge_mode" "never_enabled" {
    project_id = vercel_project.never_enabled.id
    %[2]s
}

resource "vercel_project" "enabled" {
    name = "test-acc-%[1]s-enabled"
    %[2]s
}

resource "vercel_attack_challenge_mode" "enabled" {
    project_id = vercel_project.enabled.id
    enabled = true
    %[2]s
}

data "vercel_attack_challenge_mode" "enabled" {
    project_id = vercel_attack_challenge_mode.enabled.project_id
    %[2]s
}

resource "vercel_project" "disabled" {
    name = "test-acc-%[1]s-disabled"
    %[2]s
}

resource "vercel_attack_challenge_mode" "disabled" {
    project_id = vercel_project.disabled.id
    enabled = false
    %[2]s
}

data "vercel_attack_challenge_mode" "disabled" {
    project_id = vercel_attack_challenge_mode.disabled.project_id
    %[2]s
}
`, name, teamID)
}
