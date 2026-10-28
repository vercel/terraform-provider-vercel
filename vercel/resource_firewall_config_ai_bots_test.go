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

func TestAcc_FirewallConfigAiBotsLegacyMigration(t *testing.T) {
	api := &firewallConfigLocalAPI{}
	server := httptest.NewServer(http.HandlerFunc(api.serveHTTP))
	t.Cleanup(server.Close)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"vercel": providerserver.NewProtocol6WithError(&firewallConfigLocalProvider{Provider: vercel.New(), apiURL: server.URL}),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallConfigAiBotsConfig(`action = "deny"`),
				Check:  resource.TestCheckResourceAttr("vercel_firewall_config.test", "managed_rulesets.ai_bots.action", "deny"),
			},
			{
				Config: testAccFirewallConfigAiBotsConfig(`ai_crawlers = { action = "deny" }`),
				Check: testAccFirewallConfigAiBotsCheck(api,
					client.RuleGroup{Active: true, Action: "deny"},
					client.RuleGroup{Active: true, Action: "log"},
				),
			},
		},
	})
}

func TestAcc_FirewallConfigAiBotsUnknownActionBecomesNull(t *testing.T) {
	api := &firewallConfigLocalAPI{}
	server := httptest.NewServer(http.HandlerFunc(api.serveHTTP))
	t.Cleanup(server.Close)
	config := `resource "terraform_data" "action" { input = null }` +
		testAccFirewallConfigAiBotsConfig(`action = terraform_data.action.output`)
	logGroup := client.RuleGroup{Active: true, Action: "log"}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"vercel": providerserver.NewProtocol6WithError(&firewallConfigLocalProvider{Provider: vercel.New(), apiURL: server.URL}),
		},
		Steps: []resource.TestStep{{
			Config: config,
			Check:  testAccFirewallConfigAiBotsCheck(api, logGroup, logGroup),
		}},
	})
}

func TestAcc_FirewallConfigAiBotsUnknownActionBecomesLegacy(t *testing.T) {
	api := &firewallConfigLocalAPI{}
	server := httptest.NewServer(http.HandlerFunc(api.serveHTTP))
	t.Cleanup(server.Close)
	config := `resource "terraform_data" "action" { input = "deny" }` +
		testAccFirewallConfigAiBotsConfig(`action = terraform_data.action.output`)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"vercel": providerserver.NewProtocol6WithError(&firewallConfigLocalProvider{Provider: vercel.New(), apiURL: server.URL}),
		},
		Steps: []resource.TestStep{{
			Config: config,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("vercel_firewall_config.test", "managed_rulesets.ai_bots.action", "deny"),
				resource.TestCheckNoResourceAttr("vercel_firewall_config.test", "managed_rulesets.ai_bots.ai_crawlers.action"),
				resource.TestCheckNoResourceAttr("vercel_firewall_config.test", "managed_rulesets.ai_bots.ai_assistants.action"),
			),
		}},
	})
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
