package vercel_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func getFirewallImportID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return "", fmt.Errorf("no ID is set")
		}

		if rs.Primary.Attributes["team_id"] == "" {
			return rs.Primary.ID, nil
		}
		return fmt.Sprintf("%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.Attributes["project_id"]), nil
	}
}

func TestAcc_FirewallConfigResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallConfigResource(name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.managed",
						"managed_rulesets.owasp.xss.action",
						"deny"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.managed",
						"managed_rulesets.owasp.sqli.action",
						"log"),
					resource.TestCheckNoResourceAttr(
						"vercel_firewall_config.managed",
						"managed_rulesets.owasp.sqli.active"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.managed",
						"managed_rulesets.owasp.rce.active",
						"false"),
					resource.TestCheckNoResourceAttr(
						"vercel_firewall_config.managed",
						"managed_rulesets.owasp.php"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"enabled",
						"true"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.0.name",
						"test"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.0.id",
						"rule_test"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.0.action.action",
						"deny"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.0.condition_group.0.conditions.0.type",
						"path"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.0.condition_group.0.conditions.1.value",
						"POST"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.1.id",
						"rule_test2"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.1.action.action",
						"rate_limit"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.1.action.rate_limit.limit",
						"100"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.1.action.rate_limit.algo",
						"fixed_window"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.1.action.rate_limit.action",
						"challenge"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.1.action.rate_limit.window",
						"60"),
					resource.TestCheckTypeSetElemAttr(
						"vercel_firewall_config.custom",
						"rules.rule.1.action.rate_limit.keys.*",
						"ip"),
					resource.TestCheckTypeSetElemAttr(
						"vercel_firewall_config.custom",
						"rules.rule.1.action.rate_limit.keys.*",
						"ja4"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.2.id",
						"rule_test3"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.2.action.redirect.location",
						"/bye"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.2.action.redirect.permanent",
						"false"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.ips",
						"ip_rules.rule.0.action",
						"deny"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.ips",
						"ip_rules.rule.0.hostname",
						"test.com"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.ips",
						"ip_rules.rule.1.ip",
						"1.2.3.4/32"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.ips",
						"ip_rules.rule.1.hostname",
						"*.test.com"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.ips",
						"ip_rules.rule.2.ip",
						"2.4.6.8"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.ips",
						"ip_rules.rule.2.hostname",
						"*"),
				),
			},
			{
				ImportState:       true,
				ResourceName:      "vercel_firewall_config.managed",
				ImportStateIdFunc: getFirewallImportID("vercel_firewall_config.managed"),
			},
			{
				ImportState:       true,
				ResourceName:      "vercel_firewall_config.custom",
				ImportStateIdFunc: getFirewallImportID("vercel_firewall_config.custom"),
			},
			{
				ImportState:       true,
				ResourceName:      "vercel_firewall_config.ips",
				ImportStateIdFunc: getFirewallImportID("vercel_firewall_config.ips"),
			},
			{
				Config: testAccFirewallConfigResourceUpdated(name, teamIDConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.managed",
						"managed_rulesets.owasp.xss.action",
						"deny"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.managed",
						"managed_rulesets.owasp.sqli.action",
						"deny"),
					resource.TestCheckNoResourceAttr(
						"vercel_firewall_config.managed",
						"managed_rulesets.owasp.sqli.active"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.managed",
						"managed_rulesets.owasp.rce.active",
						"false"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.managed",
						"managed_rulesets.owasp.php.action",
						"log"),
					resource.TestCheckNoResourceAttr(
						"vercel_firewall_config.managed",
						"managed_rulesets.owasp.php.active"),
					resource.TestCheckNoResourceAttr(
						"vercel_firewall_config.managed",
						"managed_rulesets.owasp.java"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"enabled",
						"true"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.0.name",
						"test1"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.0.id",
						"rule_test1"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.0.action.action",
						"deny"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.0.condition_group.0.conditions.0.type",
						"path"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.0.condition_group.0.conditions.1.value",
						"POST"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.1.id",
						"rule_test2"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.1.action.action",
						"rate_limit"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.1.action.rate_limit.limit",
						"150"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.1.action.rate_limit.algo",
						"fixed_window"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.1.action.rate_limit.action",
						"challenge"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.1.action.rate_limit.window",
						"60"),
					resource.TestCheckTypeSetElemAttr(
						"vercel_firewall_config.custom",
						"rules.rule.1.action.rate_limit.keys.*",
						"ip"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.2.id",
						"rule_test3"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.2.action.redirect.location",
						"/bye"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.custom",
						"rules.rule.2.action.redirect.permanent",
						"false"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.ips",
						"ip_rules.rule.0.action",
						"deny"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.ips",
						"ip_rules.rule.0.hostname",
						"test.com"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.ips",
						"ip_rules.rule.1.ip",
						"1.2.3.4/32"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.ips",
						"ip_rules.rule.1.hostname",
						"*.test.com"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.ips",
						"ip_rules.rule.2.ip",
						"2.4.6.8/32"),
					resource.TestCheckResourceAttr(
						"vercel_firewall_config.ips",
						"ip_rules.rule.2.hostname",
						"*"),
				),
			},
		},
	})
}

func testAccFirewallConfigResource(name, teamID string) string {
	return fmt.Sprintf(`
resource "vercel_project" "managed" {
    name = "test-acc-%[1]s-mrs"
    %[2]s
}

resource "vercel_firewall_config" "managed" {
    project_id = vercel_project.managed.id
    %[2]s

    managed_rulesets {
        owasp {
            xss = { action = "deny" }
            sqli = { action = "log" }

            rce = { action = "deny", active = false }
        }
    }
}

resource "vercel_project" "custom" {
    name = "test-acc-%[1]s-custom"
    %[2]s
}

resource "vercel_firewall_config" "custom" {
    project_id = vercel_project.custom.id
    %[2]s

    rules {
        rule {
          name =  "test"
          action = {
            action = "deny"
          }
          condition_group = [{
            conditions = [{
                type = "path"
                op = "eq"
                value = "/test"
            },
            {
              type = "method"
              op = "eq"
              value = "POST"
            }]
          }]
        }
        rule {
          name =  "test2"
          action = {
            action = "rate_limit"
            rate_limit = {
                limit = 100
                window = 60
                algo = "fixed_window"
                keys = ["ip", "ja4"]
                action = "challenge"

            }
          }
          condition_group = [{
            conditions = [{
                type = "path"
                op = "eq"
                value = "/api"
            }]
          }]
        }
        rule {
          name =  "test3"
          action = {
            action = "redirect"
            redirect = {
                location = "/bye"
                permanent = false

            }
          }
          condition_group = [{
            conditions = [{
                type = "path"
                op = "eq"
                value = "/api"
            }]
          }]
        }

        rule {
          name =  "test4"
          action = {
            action = "log"
          }
          condition_group = [{
            conditions = [{
                type = "ja4_digest"
                op = "eq"
                value = "fakeja4"
            }]
          }]
        }
    }
}

resource "vercel_project" "ips" {
    name = "test-acc-%[1]s-ips"
    %[2]s
}

resource "vercel_firewall_config" "ips" {
    project_id = vercel_project.ips.id
    %[2]s

    ip_rules {
        rule {
            action = "deny"
            ip = "5.5.0.0/16"
            hostname = "test.com"
        }
        rule {
            action = "deny"
            ip = "1.2.3.4/32"
            hostname = "*.test.com"
        }
        rule {
            action = "deny"
            ip = "2.4.6.8"
            hostname = "*"
        }
    }
}

`, name, teamID)
}

func testAccFirewallConfigResourceUpdated(name, teamID string) string {
	return fmt.Sprintf(`
resource "vercel_project" "managed" {
    name = "test-acc-%[1]s-mrs"
    %[2]s
}

resource "vercel_firewall_config" "managed" {
    project_id = vercel_project.managed.id
    %[2]s

    managed_rulesets {
        owasp {
            xss = { action = "deny", active = false }
            sqli = { action = "deny" }

            rce = { action = "deny", active = false }
            php = { action = "log" }
        }
    }
}

resource "vercel_project" "custom" {
    name = "test-acc-%[1]s-custom"
    %[2]s
}

resource "vercel_firewall_config" "custom" {
    project_id = vercel_project.custom.id
    %[2]s

    rules {
        rule {
          name =  "test1"
          action = {
            action = "deny"
          }
          condition_group = [{
            conditions = [{
                type = "path"
                op = "eq"
                value = "/test"
            },
            {
              type = "method"
              op = "eq"
              value = "POST"
            }]
          }]
        }
        rule {
          name =  "test2"
          action = {
            action = "rate_limit"
            rate_limit = {
                limit = 150
                window = 60
                algo = "fixed_window"
                keys = ["ip"]
                action = "challenge"

            }
          }
          condition_group = [{
            conditions = [{
                type = "path"
                op = "eq"
                value = "/api"
            }]
          }]
        }
        rule {
          name =  "test3"
          action = {
            action = "redirect"
            redirect = {
                location = "/bye"
                permanent = false

            }
          }
          condition_group = [{
            conditions = [{
                type = "path"
                op = "eq"
                value = "/api"
            }]
          }]
        }
    }
}

resource "vercel_project" "ips" {
    name = "test-acc-%[1]s-ips"
    %[2]s
}

resource "vercel_firewall_config" "ips" {
    project_id = vercel_project.ips.id
    %[2]s

    ip_rules {
        rule {
            action = "deny"
            ip = "5.6.0.0/16"
            hostname = "test.com"
        }
        rule {
            action = "challenge"
            ip = "1.2.3.4/32"
            hostname = "*.test.com"
        }
        rule {
            action = "deny"
            ip = "2.4.6.8/32"
            hostname = "*"
        }
    }
}`, name, teamID)
}
