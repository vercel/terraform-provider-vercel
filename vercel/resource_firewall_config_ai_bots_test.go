package vercel_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vercel/terraform-provider-vercel/v5/client"
	"github.com/vercel/terraform-provider-vercel/v5/vercel"
)

func TestAcc_FirewallConfigAiBotsLifecycle(t *testing.T) {
	api := &firewallConfigLocalAPI{}
	server := httptest.NewServer(http.HandlerFunc(api.serveHTTP))
	t.Cleanup(server.Close)
	logGroup := client.RuleGroup{Active: true, Action: "log"}
	denyGroup := client.RuleGroup{Active: true, Action: "deny"}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"vercel": providerserver.NewProtocol6WithError(&firewallConfigLocalProvider{Provider: vercel.New(), apiURL: server.URL}),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallConfigAiBotsConfig(""),
				Check:  testAccFirewallConfigAiBotsCheck(api, logGroup, logGroup),
			},
			{
				Config: testAccFirewallConfigAiBotsConfig(`ai_crawlers = { action = "deny" }`),
				Check:  testAccFirewallConfigAiBotsCheck(api, denyGroup, logGroup),
			},
			{
				PreConfig: func() {
					api.mu.Lock()
					defer api.mu.Unlock()
					api.active.ManagedRulesets["ai_bots"].RuleGroups["ai_crawlers"] = client.RuleGroup{Active: false, Action: "deny"}
				},
				Config: testAccFirewallConfigAiBotsConfig(`ai_crawlers = { action = "deny" }`),
				Check:  testAccFirewallConfigAiBotsCheck(api, denyGroup, logGroup),
			},
			{
				PreConfig: func() {
					api.mu.Lock()
					defer api.mu.Unlock()
					api.active.ManagedRulesets["ai_bots"].RuleGroups["ai_assistants"] = client.RuleGroup{Active: false, Action: "deny"}
				},
				Config: testAccFirewallConfigAiBotsConfig(`ai_crawlers = { action = "deny" }`),
				Check:  testAccFirewallConfigAiBotsCheck(api, denyGroup, logGroup),
			},
			{
				Config: testAccFirewallConfigAiBotsConfig(`ai_crawlers = { active = true, action = "log" }
ai_assistants = { active = false, action = "log" }`),
				Check: testAccFirewallConfigAiBotsCheck(api, logGroup, client.RuleGroup{Active: false, Action: "log"}),
			},
			{
				ResourceName: "vercel_firewall_config.test", ImportState: true, ImportStateVerify: true,
				ImportStateIdFunc: getFirewallImportID("vercel_firewall_config.test"),
			},
			{
				Config: testAccFirewallConfigAiBotsConfig(`ai_crawlers = { action = "deny" }`),
				Check:  testAccFirewallConfigAiBotsCheck(api, denyGroup, logGroup),
			},
		},
	})
}

func TestAcc_FirewallConfigAiBotsActionTransitions(t *testing.T) {
	for _, tc := range []struct {
		name, initial, dependency string
		legacy                    bool
	}{
		{name: "legacy_to_categories", initial: `action = "deny"`, legacy: true},
		{name: "unknown_to_null", initial: `action = terraform_data.action.output`, dependency: `resource "terraform_data" "action" { input = null }`},
		{name: "unknown_to_legacy", initial: `action = terraform_data.action.output`, dependency: `resource "terraform_data" "action" { input = "deny" }`, legacy: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := &firewallConfigLocalAPI{}
			server := httptest.NewServer(http.HandlerFunc(api.serveHTTP))
			t.Cleanup(server.Close)
			logGroup := client.RuleGroup{Active: true, Action: "log"}
			initialCheck := testAccFirewallConfigAiBotsCheck(api, logGroup, logGroup)
			if tc.legacy {
				initialCheck = resource.TestCheckResourceAttr("vercel_firewall_config.test", "managed_rulesets.ai_bots.action", "deny")
			}
			steps := []resource.TestStep{{
				Config: tc.dependency + testAccFirewallConfigAiBotsConfig(tc.initial),
				Check:  initialCheck,
			}}
			if tc.legacy {
				steps = append(steps, resource.TestStep{
					Config: tc.dependency + testAccFirewallConfigAiBotsConfig(`ai_crawlers = { action = "deny" }`),
					Check:  testAccFirewallConfigAiBotsCheck(api, client.RuleGroup{Active: true, Action: "deny"}, logGroup),
				})
			}
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
					"vercel": providerserver.NewProtocol6WithError(&firewallConfigLocalProvider{Provider: vercel.New(), apiURL: server.URL}),
				},
				Steps: steps,
			})
		})
	}
}

func testAccFirewallConfigAiBotsConfig(categories string) string {
	return fmt.Sprintf(`
resource "vercel_firewall_config" "test" {
  project_id = "prj_123"
  enabled = true
  managed_rulesets {
    ai_bots {
      active = true
      %s
    }
  }
}
`, categories)
}

func testAccFirewallConfigAiBotsCheck(api *firewallConfigLocalAPI, crawlers, assistants client.RuleGroup) resource.TestCheckFunc {
	return func(*terraform.State) error {
		api.mu.Lock()
		defer api.mu.Unlock()
		want := client.ManagedRule{Active: true, RuleGroups: map[string]client.RuleGroup{
			"ai_crawlers": crawlers, "ai_assistants": assistants,
		}}
		if got := api.active.ManagedRulesets["ai_bots"]; !reflect.DeepEqual(got, want) {
			return fmt.Errorf("AI Bots config:\ngot: %+v\nwant: %+v", got, want)
		}
		return nil
	}
}
