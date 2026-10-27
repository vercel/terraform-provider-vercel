package vercel_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vercel/terraform-provider-vercel/v5/client"
	"github.com/vercel/terraform-provider-vercel/v5/vercel"
)

func TestAcc_FirewallConfigAiBotsCategories(t *testing.T) {
	for _, tc := range []struct {
		name, initial, dependency string
		legacy                    bool
	}{
		{name: "categories"},
		{name: "legacy", initial: `action = "deny"`, legacy: true},
		{name: "unknown_to_null", initial: `action = terraform_data.action.output`, dependency: `resource "terraform_data" "action" { input = null }`},
		{name: "unknown_to_legacy", initial: `action = terraform_data.action.output`, dependency: `resource "terraform_data" "action" { input = "deny" }`, legacy: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := &firewallConfigLocalAPI{}
			server := httptest.NewServer(http.HandlerFunc(api.serveHTTP))
			t.Cleanup(server.Close)
			config := func(categories string, enabled bool) string {
				return tc.dependency + fmt.Sprintf(`
resource "vercel_firewall_config" "test" {
  project_id = "prj_123"
  enabled = %t
  managed_rulesets {
    ai_bots {
      active = true
      %s
    }
  }
}
`, enabled, categories)
			}
			checkAPI := func(crawlers, assistants client.CoreRuleSet) resource.TestCheckFunc {
				return func(*terraform.State) error {
					api.mu.Lock()
					defer api.mu.Unlock()
					rule := api.active.ManagedRulesets["ai_bots"]
					if rule.Action != "" || len(rule.RuleGroups) != 2 || rule.RuleGroups["ai_crawlers"] != crawlers || rule.RuleGroups["ai_assistants"] != assistants {
						return fmt.Errorf("unexpected AI Bots API config: %+v", rule)
					}
					return nil
				}
			}
			initialCheck := checkAPI(client.CoreRuleSet{Active: true, Action: "log"}, client.CoreRuleSet{Active: true, Action: "log"})
			if tc.legacy {
				initialCheck = resource.TestCheckResourceAttr("vercel_firewall_config.test", "managed_rulesets.ai_bots.action", "deny")
			}
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
					"vercel": providerserver.NewProtocol6WithError(&firewallConfigLocalProvider{Provider: vercel.New(), apiURL: server.URL}),
				},
				Steps: []resource.TestStep{
					{Config: config(tc.initial, true), Check: initialCheck},
					{
						Config: config(`ai_crawlers = { action = "deny" }`, true),
						Check:  checkAPI(client.CoreRuleSet{Active: true, Action: "deny"}, client.CoreRuleSet{Active: true, Action: "log"}),
					},
					{
						PreConfig: func() {
							api.mu.Lock()
							defer api.mu.Unlock()
							api.active.ManagedRulesets["ai_bots"].RuleGroups["ai_assistants"] = client.CoreRuleSet{Active: false, Action: "deny"}
						},
						Config: config(`ai_crawlers = { action = "deny" }`, false),
						Check:  checkAPI(client.CoreRuleSet{Active: true, Action: "deny"}, client.CoreRuleSet{Active: false, Action: "deny"}),
					},
					{
						Config: config(`ai_crawlers = { action = "log" }
ai_assistants = { active = false }`, false),
						Check: checkAPI(client.CoreRuleSet{Active: true, Action: "log"}, client.CoreRuleSet{Active: false, Action: "log"}),
					},
					{ResourceName: "vercel_firewall_config.test", ImportState: true, ImportStateVerify: true, ImportStateIdFunc: getFirewallImportID("vercel_firewall_config.test")},
				},
			})
		})
	}
}
