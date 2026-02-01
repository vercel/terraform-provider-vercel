package vercel_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_AttackChallengeModeDataSource(t *testing.T) {
	name := acctest.RandString(16)
	activeUntil := time.Now().Add(1 * time.Hour).UnixMilli()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccAttackChallengeModeConfig(name, activeUntil)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.vercel_attack_challenge_mode.never_enabled", "enabled", "false"),
					resource.TestCheckResourceAttr("data.vercel_attack_challenge_mode.enabled", "enabled", "true"),
					resource.TestCheckResourceAttr("data.vercel_attack_challenge_mode.enabled", "attack_mode_active_until", strconv.FormatInt(activeUntil, 10)),
					resource.TestCheckResourceAttr("data.vercel_attack_challenge_mode.disabled", "enabled", "false"),
				),
			},
		},
	})
}

func testAccAttackChallengeModeConfig(name string, activeUntil int64) string {
	return fmt.Sprintf(`
resource "vercel_project" "never_enabled" {
    name = "test-acc-%[1]s"
}

data "vercel_attack_challenge_mode" "never_enabled" {
    project_id = vercel_project.never_enabled.id
}

resource "vercel_project" "enabled" {
    name = "test-acc-%[1]s-enabled"
}

resource "vercel_attack_challenge_mode" "enabled" {
    project_id = vercel_project.enabled.id
    enabled = true
    attack_mode_active_until = %[2]d
}

data "vercel_attack_challenge_mode" "enabled" {
    project_id = vercel_attack_challenge_mode.enabled.project_id
}

resource "vercel_project" "disabled" {
    name = "test-acc-%[1]s-disabled"
}

resource "vercel_attack_challenge_mode" "disabled" {
    project_id = vercel_project.disabled.id
    enabled = false
    attack_mode_active_until = %[2]d
}

data "vercel_attack_challenge_mode" "disabled" {
    project_id = vercel_attack_challenge_mode.disabled.project_id
}
`, name, activeUntil)
}
