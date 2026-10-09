package vercel_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/vercel/terraform-provider-vercel/v5/client"
	"github.com/vercel/terraform-provider-vercel/v5/vercel"
)

type firewallConfigLocalProvider struct {
	provider.Provider
	apiURL string
}

func (p *firewallConfigLocalProvider) Configure(_ context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	apiClient := client.New("local-firewall-test").WithBaseURL(p.apiURL).WithTeam(client.Team{ID: "team_123"})
	resp.ResourceData = apiClient
	resp.DataSourceData = apiClient
}

func TestAcc_FirewallConfigUpdateRuleIDConsistency(t *testing.T) {
	tests := []struct {
		name    string
		setting string
		value   string
		rules   string
		writes  []string
		unknown bool
	}{
		{"managed_only", "managed", `"challenge"`, "", []string{"PUT"}, false},
		{"managed_and_insert", "managed", `"challenge"`, "insert", []string{"PUT"}, false},
		{"enabled_only", "enabled", "false", "", []string{"PUT"}, false},
		{"crs_only", "crs", `"deny"`, "", []string{"PUT"}, false},
		{"ip_only", "ip", `"updated notes"`, "", []string{"PUT"}, false},
		{"rules_only_complete_edit", "", "", "edit", []string{"PATCH rules.update"}, false},
		{"rules_only_edit_and_insert", "", "", "edit_insert", []string{"PATCH rules.update", "PATCH rules.insert"}, false},
		{"unknown_enabled", "enabled", "false", "", []string{"PUT"}, true},
		{"unknown_managed", "managed", `"challenge"`, "", []string{"PUT"}, true},
		{"unknown_crs", "crs", `"deny"`, "", []string{"PUT"}, true},
		{"unknown_ip", "ip", `"updated notes"`, "", []string{"PUT"}, true},
		{"unknown_settings_rules_only", "enabled", "true", "edit", []string{"PATCH rules.update"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := &firewallConfigLocalAPI{}
			server := httptest.NewServer(http.HandlerFunc(api.serveHTTP))
			t.Cleanup(server.Close)
			var ruleID, ipID string
			initial := testAccFirewallConfigUpdateConfig("", "", "", tc.unknown)
			updated := testAccFirewallConfigUpdateConfig(tc.setting, tc.value, tc.rules, tc.unknown)
			var preApply []plancheck.PlanCheck
			if tc.unknown {
				settingPaths := map[string]tfjsonpath.Path{
					"enabled": tfjsonpath.New("enabled"),
					"managed": tfjsonpath.New("managed_rulesets").AtMapKey("bot_protection").AtMapKey("action"),
					"crs":     tfjsonpath.New("managed_rulesets").AtMapKey("owasp").AtMapKey("xss").AtMapKey("action"),
					"ip":      tfjsonpath.New("ip_rules").AtMapKey("rule").AtSliceIndex(0).AtMapKey("notes"),
				}
				preApply = append(preApply, plancheck.ExpectUnknownValue("vercel_firewall_config.test", settingPaths[tc.setting]))
			}
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
					"vercel": providerserver.NewProtocol6WithError(&firewallConfigLocalProvider{Provider: vercel.New(), apiURL: server.URL}),
				},
				CheckDestroy: func(_ *terraform.State) error {
					api.mu.Lock()
					defer api.mu.Unlock()
					if len(api.active.Rules) != 0 || len(api.active.IPRules) != 0 {
						return fmt.Errorf("firewall rules remain after destroy")
					}
					return nil
				},
				Steps: []resource.TestStep{
					{
						Config: initial,
						Check: resource.ComposeAggregateTestCheckFunc(
							captureResourceAttr("vercel_firewall_config.test", "rules.rule.0.id", &ruleID),
							captureResourceAttr("vercel_firewall_config.test", "ip_rules.rule.0.id", &ipID),
						),
					},
					{
						PreConfig: func() {
							api.mu.Lock()
							defer api.mu.Unlock()
							api.writes = nil
						},
						Config:           updated,
						ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: preApply},
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttrPtr("vercel_firewall_config.test", "ip_rules.rule.0.id", &ipID),
							func(state *terraform.State) error {
								api.mu.Lock()
								defer api.mu.Unlock()
								if !reflect.DeepEqual(api.writes, tc.writes) {
									return fmt.Errorf("writes = %v, want %v", api.writes, tc.writes)
								}
								attrs := state.RootModule().Resources["vercel_firewall_config.test"].Primary.Attributes
								for i, rule := range api.active.Rules {
									if got := attrs[fmt.Sprintf("rules.rule.%d.id", i)]; got != rule.ID {
										return fmt.Errorf("state rule ID = %q, API ID = %q", got, rule.ID)
									}
								}
								if tc.writes[0] == "PUT" && attrs["rules.rule.0.id"] == ruleID {
									return fmt.Errorf("PUT did not adopt the API's new rule ID")
								}
								if tc.writes[0] != "PUT" && attrs["rules.rule.0.id"] != ruleID {
									return fmt.Errorf("PATCH changed the existing rule ID")
								}
								return nil
							},
						),
					},
					{Config: updated, PlanOnly: true},
				},
			})
		})
	}
}

func testAccFirewallConfigUpdateConfig(setting, value, ruleChange string, deferred bool) string {
	settings := map[string]string{"enabled": "true", "managed": `"log"`, "crs": `"log"`, "ip": `""`}
	if setting != "" {
		settings[setting] = value
	}
	var dependencies string
	if deferred {
		dependencies = fmt.Sprintf(`
resource "terraform_data" "settings" {
  input = { enabled = %s, managed = %s, crs = %s, ip = %s, rule_change = %q }
}
`, settings["enabled"], settings["managed"], settings["crs"], settings["ip"], ruleChange)
		for key := range settings {
			settings[key] = "terraform_data.settings.output." + key
		}
	}
	rule := func(name, path, action string) string {
		return fmt.Sprintf(`
    rule {
      name = %q
      active = true
      action = { action = %q }
      condition_group = [{ conditions = [{ type = "path", op = "eq", value = %q }] }]
    }
`, name, action, path)
	}
	rules := rule("Block scanner probes", "/wp-admin", "deny")
	if ruleChange == "edit" || ruleChange == "edit_insert" {
		rules = rule("Renamed rule", "/changed", "challenge")
	}
	if ruleChange == "insert" || ruleChange == "edit_insert" {
		rules += rule("Block second path", "/scanner", "deny")
	}
	return fmt.Sprintf(`
%s
resource "vercel_firewall_config" "test" {
  project_id = "prj_123"
  enabled = %s
  managed_rulesets {
    bot_protection {
      active = true
      action = %s
    }
    owasp { xss = { active = false, action = %s } }
  }
  ip_rules {
    rule {
      hostname = "example.com"
      ip = "192.0.2.1"
      action = "deny"
      notes = %s
    }
  }
  rules { %s }
}
`, dependencies, settings["enabled"], settings["managed"], settings["crs"], settings["ip"], rules)
}

type firewallConfigLocalAPI struct {
	mu     sync.Mutex
	active client.FirewallConfig
	writes []string
	serial int
}

func (api *firewallConfigLocalAPI) serveHTTP(w http.ResponseWriter, req *http.Request) {
	api.mu.Lock()
	defer api.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if req.Method == http.MethodGet && req.URL.Path == "/v1/security/firewall/config/active" {
		_ = json.NewEncoder(w).Encode(api.active)
		return
	}
	if req.URL.Path != "/v1/security/firewall/config" {
		http.Error(w, "unexpected endpoint", http.StatusNotFound)
		return
	}
	switch req.Method {
	case http.MethodPut:
		var incoming client.FirewallConfig
		if err := json.NewDecoder(req.Body).Decode(&incoming); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		api.serial++
		// Full PUT regenerates custom IDs, while matching IP rules retain their IDs.
		for i := range incoming.Rules {
			incoming.Rules[i].ID = fmt.Sprintf("rule_put_%d_%d", api.serial, i)
		}
		for i := range incoming.IPRules {
			incoming.IPRules[i].ID = fmt.Sprintf("ip_%d_%d", api.serial, i)
			for _, previous := range api.active.IPRules {
				if previous.Hostname == incoming.IPRules[i].Hostname && previous.IP == incoming.IPRules[i].IP {
					incoming.IPRules[i].ID = previous.ID
				}
			}
		}
		api.active = incoming
		api.writes = append(api.writes, "PUT")
		_ = json.NewEncoder(w).Encode(map[string]any{"active": api.active})
	case http.MethodPatch:
		var patch struct {
			Action string              `json:"action"`
			ID     string              `json:"id"`
			Value  client.FirewallRule `json:"value"`
		}
		if err := json.NewDecoder(req.Body).Decode(&patch); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		switch patch.Action {
		case "rules.update":
			found := false
			for i, rule := range api.active.Rules {
				if rule.ID == patch.ID {
					patch.Value.ID = rule.ID
					api.active.Rules[i] = patch.Value
					found = true
				}
			}
			if !found {
				http.Error(w, "rule not found", http.StatusNotFound)
				return
			}
		case "rules.insert":
			api.serial++
			patch.Value.ID = fmt.Sprintf("rule_insert_%d", api.serial)
			api.active.Rules = append(api.active.Rules, patch.Value)
		default:
			http.Error(w, "unexpected PATCH action", http.StatusBadRequest)
			return
		}
		api.writes = append(api.writes, "PATCH "+patch.Action)
		_, _ = w.Write([]byte(`{}`))
	default:
		http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
	}
}
