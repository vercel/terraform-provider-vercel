package vercel

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func aiBotsRuleGroups(crawlers, assistants client.RuleGroup) map[string]client.RuleGroup {
	return map[string]client.RuleGroup{"ai_crawlers": crawlers, "ai_assistants": assistants}
}

func TestAiBotsToClient(t *testing.T) {
	logGroup := client.RuleGroup{Active: true, Action: "log"}
	tests := []struct {
		name   string
		config AiBotsConfig
		want   client.ManagedRule
	}{
		{
			name:   "no categories",
			config: AiBotsConfig{Active: types.BoolValue(true)},
			want:   client.ManagedRule{Active: true},
		},
		{
			name:   "omitted assistants",
			config: AiBotsConfig{Active: types.BoolValue(true), AiCrawlers: &RuleGroup{Action: types.StringValue("deny")}},
			want:   client.ManagedRule{Active: true, RuleGroups: aiBotsRuleGroups(client.RuleGroup{Active: true, Action: "deny"}, logGroup)},
		},
		{
			name: "disabled assistants",
			config: AiBotsConfig{
				Active:       types.BoolValue(true),
				AiAssistants: &RuleGroup{Active: types.BoolValue(false), Action: types.StringValue("deny")},
			},
			want: client.ManagedRule{Active: true, RuleGroups: aiBotsRuleGroups(logGroup, client.RuleGroup{Active: false, Action: "deny"})},
		},
		{
			name:   "disabled ruleset",
			config: AiBotsConfig{Active: types.BoolValue(false)},
			want:   client.ManagedRule{Active: false},
		},
		{
			name:   "legacy action",
			config: AiBotsConfig{Active: types.BoolValue(true), Action: types.StringValue("challenge")},
			want:   client.ManagedRule{Active: true, Action: "challenge"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.config.toClient(); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("toClient():\ngot: %+v\nwant: %+v", got, tc.want)
			}
		})
	}
}

func TestFromAiBots(t *testing.T) {
	logGroup := client.RuleGroup{Active: true, Action: "log"}
	tests := []struct {
		name string
		rule client.ManagedRule
		ref  *AiBotsConfig
		want *AiBotsConfig
	}{
		{
			name: "preserves omitted values",
			rule: client.ManagedRule{Active: false, RuleGroups: aiBotsRuleGroups(client.RuleGroup{Active: true, Action: "deny"}, logGroup)},
			ref:  &AiBotsConfig{AiCrawlers: &RuleGroup{Action: types.StringValue("deny")}},
			want: &AiBotsConfig{AiCrawlers: &RuleGroup{Action: types.StringValue("deny")}},
		},
		{
			name: "preserves explicit defaults",
			rule: client.ManagedRule{Active: false, RuleGroups: aiBotsRuleGroups(logGroup, logGroup)},
			ref: &AiBotsConfig{
				Active:     types.BoolValue(false),
				AiCrawlers: &RuleGroup{Active: types.BoolValue(true), Action: types.StringValue("log")},
			},
			want: &AiBotsConfig{
				Active:     types.BoolValue(false),
				AiCrawlers: &RuleGroup{Active: types.BoolValue(true), Action: types.StringValue("log")},
			},
		},
		{
			name: "preserves legacy action",
			rule: client.ManagedRule{Active: true, Action: "deny"},
			ref:  &AiBotsConfig{Active: types.BoolValue(true), Action: types.StringValue("deny")},
			want: &AiBotsConfig{Active: types.BoolValue(true), Action: types.StringValue("deny")},
		},
		{
			name: "detects legacy migration",
			rule: client.ManagedRule{Active: true, RuleGroups: aiBotsRuleGroups(client.RuleGroup{Active: true, Action: "deny"}, logGroup)},
			ref:  &AiBotsConfig{Active: types.BoolValue(true), Action: types.StringValue("log")},
			want: &AiBotsConfig{
				Active:     types.BoolValue(true),
				AiCrawlers: &RuleGroup{Active: types.BoolValue(true), Action: types.StringValue("deny")},
			},
		},
		{
			name: "detects drift",
			rule: client.ManagedRule{Active: true, RuleGroups: aiBotsRuleGroups(
				client.RuleGroup{Active: false, Action: "deny"},
				client.RuleGroup{Active: false, Action: "deny"},
			)},
			ref: &AiBotsConfig{AiCrawlers: &RuleGroup{Action: types.StringValue("log")}},
			want: &AiBotsConfig{
				Active:       types.BoolValue(true),
				AiCrawlers:   &RuleGroup{Active: types.BoolValue(false), Action: types.StringValue("deny")},
				AiAssistants: &RuleGroup{Active: types.BoolValue(false), Action: types.StringValue("deny")},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := fromAiBots(tc.rule, tc.ref, preserveConfiguredShape); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("fromAiBots():\ngot: %+v\nwant: %+v", got, tc.want)
			}
		})
	}
}

func TestCanonicalImportAiBots(t *testing.T) {
	rule := client.ManagedRule{Active: true, Action: "deny", RuleGroups: aiBotsRuleGroups(
		client.RuleGroup{Active: true, Action: "deny"},
		client.RuleGroup{Active: false, Action: "log"},
	)}
	got := importFirewallConfig(t, client.FirewallConfig{ManagedRulesets: map[string]client.ManagedRule{"ai_bots": rule}})
	want := &AiBotsConfig{
		Active:       types.BoolValue(true),
		AiCrawlers:   &RuleGroup{Active: types.BoolValue(true), Action: types.StringValue("deny")},
		AiAssistants: &RuleGroup{Active: types.BoolValue(false), Action: types.StringValue("log")},
	}
	if !reflect.DeepEqual(got.ManagedRulesets.AiBots, want) {
		t.Fatalf("imported config:\ngot: %+v\nwant: %+v", got.ManagedRulesets.AiBots, want)
	}
}
