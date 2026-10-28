package vercel_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/vercel/terraform-provider-vercel/v5/client"
	"github.com/vercel/terraform-provider-vercel/v5/vercel"
)

var (
	activeLog  = client.RuleGroup{Active: true, Action: "log"}
	activeDeny = client.RuleGroup{Active: true, Action: "deny"}
)

func TestFirewallConfigAiBotsRejectsInvalidConfig(t *testing.T) {
	_, providers := newFakeFirewallAPI(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providers,
		Steps: []resource.TestStep{
			{
				Config:      createAiBotsConfig(`ai_crawlers = {}`),
				ExpectError: regexp.MustCompile(`attribute "ai_crawlers": attribute "action" is\s+required`),
			},
			{
				Config:      createAiBotsConfig(`ai_assistants = {}`),
				ExpectError: regexp.MustCompile(`attribute "ai_assistants": attribute "action" is\s+required`),
			},
			{
				Config:      createAiBotsConfig(`ai_crawlers = { action = "challenge" }`),
				ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Value Match.*ai_crawlers\.action`),
			},
			{
				Config:      createAiBotsConfig(`ai_assistants = { action = "allow" }`),
				ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Value Match.*ai_assistants\.action`),
			},
			{
				Config: createAiBotsConfig(`
      action      = "deny"
      ai_crawlers = { action = "log" }`),
				ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Combination.*ai_crawlers`),
			},
			{
				Config: createAiBotsConfig(`
      action        = "deny"
      ai_assistants = { action = "log" }`),
				ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Combination.*ai_assistants`),
			},
		},
	})
}

func TestAcc_FirewallConfigAiBotsLifecycle(t *testing.T) {
	api, providers := newFakeFirewallAPI(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providers,
		Steps: []resource.TestStep{
			{
				Config: createAiBotsConfig(""),
				Check:  expectAiBotsInAPI(api, aiBotsCategories{crawlers: activeLog, assistants: activeLog}),
			},
			{
				Config: createAiBotsConfig(`ai_crawlers = { action = "deny" }`),
				Check:  expectAiBotsInAPI(api, aiBotsCategories{crawlers: activeDeny, assistants: activeLog}),
			},
			{
				Config: createAiBotsConfig(`
      ai_crawlers   = { active = true, action = "log" }
      ai_assistants = { active = false, action = "log" }`),
				Check: expectAiBotsInAPI(api, aiBotsCategories{crawlers: activeLog, assistants: client.RuleGroup{Active: false, Action: "log"}}),
			},
			{
				ResourceName:      "vercel_firewall_config.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: getFirewallImportID("vercel_firewall_config.test"),
			},
			{
				// Removing ai_assistants from the config resets it to the default
				// instead of keeping the disabled setting from the previous step.
				Config: createAiBotsConfig(`ai_crawlers = { action = "deny" }`),
				Check:  expectAiBotsInAPI(api, aiBotsCategories{crawlers: activeDeny, assistants: activeLog}),
			},
		},
	})
}

// Each step changes a category outside of Terraform, and the apply has to put
// back what the config says.
func TestAcc_FirewallConfigAiBotsDrift(t *testing.T) {
	api, providers := newFakeFirewallAPI(t)
	config := createAiBotsConfig(`ai_crawlers = { action = "deny" }`)
	want := aiBotsCategories{crawlers: activeDeny, assistants: activeLog}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providers,
		Steps: []resource.TestStep{
			{Config: config, Check: expectAiBotsInAPI(api, want)},
			{
				PreConfig: changeCategoryInAPI(api, "ai_crawlers", client.RuleGroup{Active: false, Action: "deny"}),
				Config:    config,
				Check:     expectAiBotsInAPI(api, want),
			},
			{
				// ai_assistants isn't in the config, so drift on it is reset to the default.
				PreConfig: changeCategoryInAPI(api, "ai_assistants", client.RuleGroup{Active: false, Action: "deny"}),
				Config:    config,
				Check:     expectAiBotsInAPI(api, want),
			},
		},
	})
}

func TestAcc_FirewallConfigAiBotsSwitchFromActionToCategories(t *testing.T) {
	api, providers := newFakeFirewallAPI(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providers,
		Steps: []resource.TestStep{
			{
				Config: createAiBotsConfig(`action = "deny"`),
				Check:  resource.TestCheckResourceAttr("vercel_firewall_config.test", "managed_rulesets.ai_bots.action", "deny"),
			},
			{
				Config: createAiBotsConfig(`ai_crawlers = { action = "deny" }`),
				Check:  expectAiBotsInAPI(api, aiBotsCategories{crawlers: activeDeny, assistants: activeLog}),
			},
		},
	})
}

// terraform_data.output is unknown until apply, so these tests plan with an
// unknown legacy action and check what it resolves to.
func TestAcc_FirewallConfigAiBotsNullActionFromAnotherResource(t *testing.T) {
	api, providers := newFakeFirewallAPI(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providers,
		Steps: []resource.TestStep{{
			Config: `resource "terraform_data" "action" { input = null }` +
				createAiBotsConfig(`action = terraform_data.action.output`),
			Check: expectAiBotsInAPI(api, aiBotsCategories{crawlers: activeLog, assistants: activeLog}),
		}},
	})
}

func TestAcc_FirewallConfigAiBotsActionFromAnotherResource(t *testing.T) {
	_, providers := newFakeFirewallAPI(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providers,
		Steps: []resource.TestStep{{
			Config: `resource "terraform_data" "action" { input = "deny" }` +
				createAiBotsConfig(`action = terraform_data.action.output`),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("vercel_firewall_config.test", "managed_rulesets.ai_bots.action", "deny"),
				resource.TestCheckNoResourceAttr("vercel_firewall_config.test", "managed_rulesets.ai_bots.ai_crawlers.action"),
				resource.TestCheckNoResourceAttr("vercel_firewall_config.test", "managed_rulesets.ai_bots.ai_assistants.action"),
			),
		}},
	})
}

func newFakeFirewallAPI(t *testing.T) (*firewallConfigLocalAPI, map[string]func() (tfprotov6.ProviderServer, error)) {
	api := &firewallConfigLocalAPI{}
	server := httptest.NewServer(http.HandlerFunc(api.serveHTTP))
	t.Cleanup(server.Close)
	return api, map[string]func() (tfprotov6.ProviderServer, error){
		"vercel": providerserver.NewProtocol6WithError(&firewallConfigLocalProvider{Provider: vercel.New(), apiURL: server.URL}),
	}
}

func createAiBotsConfig(categories string) string {
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

// aiBotsCategories is the AI Bots config the fake API should have received.
type aiBotsCategories struct {
	crawlers, assistants client.RuleGroup
}

func expectAiBotsInAPI(api *firewallConfigLocalAPI, want aiBotsCategories) resource.TestCheckFunc {
	return func(*terraform.State) error {
		api.mu.Lock()
		defer api.mu.Unlock()
		wantRule := client.ManagedRule{Active: true, RuleGroups: map[string]client.RuleGroup{
			"ai_crawlers": want.crawlers, "ai_assistants": want.assistants,
		}}
		if got := api.active.ManagedRulesets["ai_bots"]; !reflect.DeepEqual(got, wantRule) {
			return fmt.Errorf("AI Bots config:\ngot: %+v\nwant: %+v", got, wantRule)
		}
		return nil
	}
}

func changeCategoryInAPI(api *firewallConfigLocalAPI, category string, group client.RuleGroup) func() {
	return func() {
		api.mu.Lock()
		defer api.mu.Unlock()
		api.active.ManagedRulesets["ai_bots"].RuleGroups[category] = group
	}
}
