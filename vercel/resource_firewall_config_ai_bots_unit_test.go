package vercel

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestFirewallConfigAiBotsValidation(t *testing.T) {
	ctx := context.Background()
	server := providerserver.NewProtocol6(New())()
	schemaResp := &resource.SchemaResponse{}
	newFirewallConfigResource().Schema(ctx, resource.SchemaRequest{}, schemaResp)
	botsPath := tftypes.NewAttributePath().WithAttributeName("managed_rulesets").WithAttributeName("ai_bots")
	crawlerActionPath := botsPath.WithAttributeName("ai_crawlers").WithAttributeName("action")
	assistantActionPath := botsPath.WithAttributeName("ai_assistants").WithAttributeName("action")
	for _, tc := range []struct {
		name      string
		bots      AiBotsConfig
		errorPath *tftypes.AttributePath
	}{
		{"omitted categories", AiBotsConfig{}, nil},
		{"crawler deny", AiBotsConfig{AiCrawlers: &RuleGroup{Action: types.StringValue("deny")}}, nil},
		{"assistant log", AiBotsConfig{AiAssistants: &RuleGroup{Action: types.StringValue("log")}}, nil},
		{"separate actions", AiBotsConfig{AiCrawlers: &RuleGroup{Action: types.StringValue("log")}, AiAssistants: &RuleGroup{Active: types.BoolValue(false), Action: types.StringValue("deny")}}, nil},
		{"legacy action", AiBotsConfig{Action: types.StringValue("challenge")}, nil},
		{"unknown legacy action", AiBotsConfig{Action: types.StringUnknown()}, nil},
		{"missing crawler action", AiBotsConfig{AiCrawlers: &RuleGroup{}}, crawlerActionPath},
		{"missing assistant action", AiBotsConfig{AiAssistants: &RuleGroup{}}, assistantActionPath},
		{"crawler challenge", AiBotsConfig{AiCrawlers: &RuleGroup{Action: types.StringValue("challenge")}}, crawlerActionPath},
		{"assistant allow", AiBotsConfig{AiAssistants: &RuleGroup{Action: types.StringValue("allow")}}, assistantActionPath},
		{"invalid action", AiBotsConfig{AiCrawlers: &RuleGroup{Action: types.StringValue("invalid")}}, crawlerActionPath},
		{"legacy with crawlers", AiBotsConfig{Action: types.StringValue("deny"), AiCrawlers: &RuleGroup{Action: types.StringValue("log")}}, botsPath.WithAttributeName("action")},
		{"legacy with assistants", AiBotsConfig{Action: types.StringValue("deny"), AiAssistants: &RuleGroup{Action: types.StringValue("log")}}, botsPath.WithAttributeName("action")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := tfsdk.State{Schema: schemaResp.Schema}
			if diags := state.Set(ctx, FirewallConfig{ProjectID: types.StringValue("prj_123"), ManagedRulesets: &FirewallManagedRulesets{AiBots: &tc.bots}}); diags.HasError() {
				t.Fatal(diags)
			}
			config, err := tfprotov6.NewDynamicValue(state.Raw.Type(), state.Raw)
			if err != nil {
				t.Fatal(err)
			}
			response, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{TypeName: "vercel_firewall_config", Config: &config})
			if err != nil {
				t.Fatal(err)
			}
			var errors []tfprotov6.Diagnostic
			for _, diag := range response.Diagnostics {
				if diag.Severity == tfprotov6.DiagnosticSeverityError {
					errors = append(errors, *diag)
				}
			}
			if tc.errorPath == nil {
				if len(errors) != 0 {
					t.Fatalf("unexpected validation errors: %v", errors)
				}
			} else if len(errors) != 1 || !reflect.DeepEqual(errors[0].Attribute, tc.errorPath) {
				t.Fatalf("expected one validation error at %s, got %v", tc.errorPath, errors)
			}
		})
	}
}

func TestAiBotsToClient(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config AiBotsConfig
		want   client.ManagedRule
	}{
		{
			name:   "default categories",
			config: AiBotsConfig{Active: types.BoolValue(true)},
			want: client.ManagedRule{Active: true, RuleGroups: map[string]client.RuleGroup{
				"ai_crawlers":   {Active: true, Action: "log"},
				"ai_assistants": {Active: true, Action: "log"},
			}},
		},
		{
			name:   "one category leaves the other at log",
			config: AiBotsConfig{Active: types.BoolValue(true), AiCrawlers: &RuleGroup{Action: types.StringValue("deny")}},
			want: client.ManagedRule{Active: true, RuleGroups: map[string]client.RuleGroup{
				"ai_crawlers":   {Active: true, Action: "deny"},
				"ai_assistants": {Active: true, Action: "log"},
			}},
		},
		{
			name:   "disabled assistants category",
			config: AiBotsConfig{Active: types.BoolValue(true), AiAssistants: &RuleGroup{Active: types.BoolValue(false), Action: types.StringValue("deny")}},
			want: client.ManagedRule{Active: true, RuleGroups: map[string]client.RuleGroup{
				"ai_crawlers":   {Active: true, Action: "log"},
				"ai_assistants": {Active: false, Action: "deny"},
			}},
		},
		{
			name:   "disabled ruleset still sends categories",
			config: AiBotsConfig{Active: types.BoolValue(false)},
			want: client.ManagedRule{RuleGroups: map[string]client.RuleGroup{
				"ai_crawlers":   {Active: true, Action: "log"},
				"ai_assistants": {Active: true, Action: "log"},
			}},
		},
		{
			name:   "explicit legacy action stays backwards compatible",
			config: AiBotsConfig{Active: types.BoolValue(true), Action: types.StringValue("challenge")},
			want:   client.ManagedRule{Active: true, Action: "challenge"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := FirewallConfig{ManagedRulesets: &FirewallManagedRulesets{AiBots: &tc.config}}
			got, err := cfg.toClient()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.ManagedRulesets["ai_bots"], tc.want) {
				t.Fatalf("AI Bots = %+v, want %+v", got.ManagedRulesets["ai_bots"], tc.want)
			}
		})
	}
}

func TestAiBotsRoundTripConfiguredShape(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config AiBotsConfig
	}{
		{"omitted categories", AiBotsConfig{Active: types.BoolValue(true)}},
		{"disabled ruleset", AiBotsConfig{Active: types.BoolValue(false)}},
		{"omitted ruleset active", AiBotsConfig{}},
		{"omitted category active", AiBotsConfig{Active: types.BoolValue(true), AiCrawlers: &RuleGroup{Action: types.StringValue("deny")}}},
		{"explicit category active", AiBotsConfig{Active: types.BoolValue(true), AiCrawlers: &RuleGroup{Active: types.BoolValue(true), Action: types.StringValue("deny")}}},
		{"configured default category", AiBotsConfig{Active: types.BoolValue(true), AiAssistants: &RuleGroup{Action: types.StringValue("log")}}},
		{"disabled category", AiBotsConfig{Active: types.BoolValue(true), AiAssistants: &RuleGroup{Active: types.BoolValue(false), Action: types.StringValue("log")}}},
		{"legacy action", AiBotsConfig{Active: types.BoolValue(true), Action: types.StringValue("deny")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := fromAiBots(tc.config.toClient(), &tc.config, preserveConfiguredShape)
			if !reflect.DeepEqual(got, &tc.config) {
				t.Errorf("configured shape changed:\ngot: %+v\nwant: %+v", got, tc.config)
			}
		})
	}
}

func TestAiBotsImportCategories(t *testing.T) {
	rule := client.ManagedRule{Active: true, Action: "deny", RuleGroups: map[string]client.RuleGroup{
		"ai_crawlers":   {Active: true, Action: "deny"},
		"ai_assistants": {Active: false, Action: "log"},
	}}
	got := importFirewallConfig(t, client.FirewallConfig{ManagedRulesets: map[string]client.ManagedRule{"ai_bots": rule}})
	bots := got.ManagedRulesets.AiBots
	if !bots.Action.IsNull() || bots.AiCrawlers.Action.ValueString() != "deny" || bots.AiAssistants.Active.ValueBool() {
		t.Fatalf("imported categories = %+v", bots)
	}
	api, err := got.toClient()
	if err != nil {
		t.Fatal(err)
	}
	rule.Action = ""
	if !reflect.DeepEqual(api.ManagedRulesets["ai_bots"], rule) {
		t.Fatalf("import round trip = %+v", api.ManagedRulesets["ai_bots"])
	}
}

func TestAiBotsReadCategoryDrift(t *testing.T) {
	for _, tc := range []struct {
		name string
		ref  AiBotsConfig
	}{
		{"omitted category active", AiBotsConfig{Active: types.BoolValue(true), AiCrawlers: &RuleGroup{Action: types.StringValue("log")}}},
		{"explicit category active", AiBotsConfig{Active: types.BoolValue(true), AiCrawlers: &RuleGroup{Active: types.BoolValue(true), Action: types.StringValue("log")}}},
		{"omitted ruleset active", AiBotsConfig{}},
		{"migrated legacy action", AiBotsConfig{Active: types.BoolValue(true), Action: types.StringValue("log")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := fromAiBots(client.ManagedRule{Active: true, RuleGroups: map[string]client.RuleGroup{"ai_crawlers": {Active: false, Action: "deny"}}}, &tc.ref, preserveConfiguredShape)
			if got.AiCrawlers == nil || got.AiCrawlers.Active.ValueBool() || got.AiCrawlers.Action.ValueString() != "deny" {
				t.Fatalf("category drift not reflected: %+v", got.AiCrawlers)
			}
			if !got.Active.ValueBool() || !got.Action.IsNull() {
				t.Fatalf("ruleset drift not reflected: %+v", got)
			}
		})
	}
}
