package vercel_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAcc_FirewallBypassResource(t *testing.T) {
	name := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(testAccFirewallBypassConfigResource(name, testGithubRepo(&testing.T{}))),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_firewall_bypass.bypass_one", "domain", "test-acc-domain-"+name+".vercel.app"),
					resource.TestCheckResourceAttr("vercel_firewall_bypass.bypass_some", "domain", "*"),
					resource.TestCheckResourceAttr("vercel_firewall_bypass.bypass_one", "source_ip", "1.2.3.4"),
					resource.TestCheckResourceAttr("vercel_firewall_bypass.bypass_some", "source_ip", "2.3.4.0/24"),
					resource.TestCheckResourceAttrWith("vercel_firewall_bypass.bypass_one", "id", func(id string) error {
						if !strings.HasSuffix(id, "#test-acc-domain-"+name+".vercel.app#1.2.3.4") {
							return fmt.Errorf("expected id does not match got %s - expected %s", id, "test-acc-domain-"+name+".vercel.app#1.2.3.4")
						}
						return nil
					}),
					resource.TestCheckResourceAttrWith("vercel_firewall_bypass.bypass_some", "id", func(id string) error {
						if !strings.HasSuffix(id, "#2.3.4.0/24") {
							return fmt.Errorf("expected id does not match suffix got %s - expected %s", id, "#2.3.4.0/24")
						}
						return nil
					}),
				),
			},
			{
				ImportState:  true,
				ResourceName: "vercel_firewall_bypass.bypass_one",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["vercel_firewall_bypass.bypass_one"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.ID), nil
				},
			},
			{
				ImportState:  true,
				ResourceName: "vercel_firewall_bypass.bypass_some",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["vercel_firewall_bypass.bypass_some"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes["team_id"], rs.Primary.ID), nil
				},
			},
			{
				Config: cfg(testAccFirewallBypassConfigResourceUpdated(name, testGithubRepo(t))),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_firewall_bypass.bypass_one", "source_ip", "0.0.0.0/0"),
					resource.TestCheckResourceAttrWith("vercel_firewall_bypass.bypass_one", "id", func(id string) error {
						if !strings.HasSuffix(id, "#test-acc-domain-"+name+".vercel.app#0.0.0.0/0") {
							return fmt.Errorf("expected id does not match got %s - expected %s", id, "test-acc-domain-"+name+".vercel.app#0.0.0.0/0")
						}
						return nil
					}),
				),
			},
		},
	})
}

func testAccFirewallBypassConfigResource(name, githubRepo string) string {
	return fmt.Sprintf(`
resource "vercel_project" "bypass_project" {
  name = "test-acc-%[1]s-enabled"
  git_repository = {
    type = "github"
    repo = "%[2]s"
  }
}

resource "vercel_project_domain" "test" {
  domain = "test-acc-domain-%[1]s.vercel.app"
  project_id = vercel_project.bypass_project.id
}

resource "vercel_firewall_bypass" "bypass_one" {
  project_id = vercel_project.bypass_project.id
  domain    = vercel_project_domain.test.domain
  source_ip = "1.2.3.4"

  depends_on = [vercel_project_domain.test]
}

resource "vercel_firewall_bypass" "bypass_some" {
  project_id = vercel_project.bypass_project.id
  domain    = "*"
  source_ip = "2.3.4.0/24"

  depends_on = [vercel_project_domain.test]
}

`, name, githubRepo)
}

func testAccFirewallBypassConfigResourceUpdated(name, githubRepo string) string {
	return fmt.Sprintf(`
resource "vercel_project" "bypass_project" {
  name = "test-acc-%[1]s-enabled"
  git_repository = {
    type = "github"
    repo = "%[2]s"
  }
}

resource "vercel_project_domain" "test" {
  domain = "test-acc-domain-%[1]s.vercel.app"
  project_id = vercel_project.bypass_project.id
}

resource "vercel_firewall_bypass" "bypass_one" {
  project_id = vercel_project.bypass_project.id
  domain    = vercel_project_domain.test.domain
  source_ip = "0.0.0.0/0"

  depends_on = [vercel_project_domain.test]
}

`, name, githubRepo)
}
