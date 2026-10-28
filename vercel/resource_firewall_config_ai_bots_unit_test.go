package vercel

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestAiBotsPlanning(t *testing.T) {
	ctx := context.Background()
	server := providerserver.NewProtocol6(New())()
	for _, tc := range []struct {
		name    string
		bots    AiBotsConfig
		invalid bool
	}{
		{"default categories", AiBotsConfig{Active: types.BoolValue(true)}, false},
		{"missing crawler action", AiBotsConfig{Active: types.BoolValue(true), AiCrawlers: &CRSRuleConfig{}}, true},
		{"missing assistant action", AiBotsConfig{Active: types.BoolValue(true), AiAssistants: &CRSRuleConfig{Active: types.BoolValue(false)}}, true},
		{"separate actions", AiBotsConfig{Active: types.BoolValue(true), AiCrawlers: &CRSRuleConfig{Action: types.StringValue("deny")}, AiAssistants: &CRSRuleConfig{Active: types.BoolValue(false), Action: types.StringValue("log")}}, false},
		{"legacy action", AiBotsConfig{Active: types.BoolValue(true), Action: types.StringValue("challenge")}, false},
		{"unknown legacy action", AiBotsConfig{Active: types.BoolValue(true), Action: types.StringUnknown()}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := FirewallConfig{ProjectID: types.StringValue("prj_123"), ManagedRulesets: &FirewallManagedRulesets{AiBots: &tc.bots}}
			state, _ := firewallConfigStateAndPlan(t, &firewallConfigResource{}, cfg, cfg)
			config, err := tfprotov6.NewDynamicValue(state.Raw.Type(), state.Raw)
			if err != nil {
				t.Fatal(err)
			}
			prior, err := tfprotov6.NewDynamicValue(state.Raw.Type(), tftypes.NewValue(state.Raw.Type(), nil))
			if err != nil {
				t.Fatal(err)
			}
			validation, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{TypeName: "vercel_firewall_config", Config: &config})
			if err != nil {
				t.Fatal(err)
			}
			invalid := false
			for _, diag := range validation.Diagnostics {
				if diag.Severity == tfprotov6.DiagnosticSeverityError {
					invalid = true
				}
			}
			if invalid != tc.invalid {
				t.Fatalf("invalid config = %t, want %t: %v", invalid, tc.invalid, validation.Diagnostics)
			}
			if invalid {
				return
			}
			planned, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{TypeName: "vercel_firewall_config", Config: &config, ProposedNewState: &config, PriorState: &prior})
			if err != nil {
				t.Fatal(err)
			}
			for _, diag := range planned.Diagnostics {
				if diag.Severity == tfprotov6.DiagnosticSeverityError {
					t.Fatalf("planning: %s: %s", diag.Summary, diag.Detail)
				}
			}
			raw, err := planned.PlannedState.Unmarshal(state.Raw.Type())
			if err != nil {
				t.Fatal(err)
			}
			plan := tfsdk.Plan{Raw: raw, Schema: state.Schema}
			var decoded FirewallConfig
			if diags := plan.Get(ctx, &decoded); diags.HasError() {
				t.Fatalf("plan must decode for Create and Update: %v", diags)
			}
			for name, ref := range map[string]*CRSRuleConfig{"ai_crawlers": tc.bots.AiCrawlers, "ai_assistants": tc.bots.AiAssistants} {
				var category types.Object
				if diags := plan.GetAttribute(ctx, path.Root("managed_rulesets").AtName("ai_bots").AtName(name), &category); diags.HasError() {
					t.Fatal(diags)
				}
				if ref == nil {
					if !category.IsNull() {
						t.Fatalf("omitted %s should stay null, got %s", name, category)
					}
					continue
				}
				var got CRSRuleConfig
				if diags := category.As(ctx, &got, basetypes.ObjectAsOptions{}); diags.HasError() {
					t.Fatal(diags)
				}
				if !reflect.DeepEqual(got, *ref) {
					t.Fatalf("%s plan = %+v, want %+v", name, got, *ref)
				}
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
			want: client.ManagedRule{Active: true, RuleGroups: map[string]client.CoreRuleSet{
				"ai_crawlers":   {Active: true, Action: "log"},
				"ai_assistants": {Active: true, Action: "log"},
			}},
		},
		{
			name:   "one category leaves the other at log",
			config: AiBotsConfig{Active: types.BoolValue(true), AiCrawlers: &CRSRuleConfig{Action: types.StringValue("deny")}},
			want: client.ManagedRule{Active: true, RuleGroups: map[string]client.CoreRuleSet{
				"ai_crawlers":   {Active: true, Action: "deny"},
				"ai_assistants": {Active: true, Action: "log"},
			}},
		},
		{
			name:   "disabled category allows assistants",
			config: AiBotsConfig{Active: types.BoolValue(true), AiAssistants: &CRSRuleConfig{Active: types.BoolValue(false), Action: types.StringValue("deny")}},
			want: client.ManagedRule{Active: true, RuleGroups: map[string]client.CoreRuleSet{
				"ai_crawlers":   {Active: true, Action: "log"},
				"ai_assistants": {Active: false, Action: "deny"},
			}},
		},
		{
			name:   "disabled ruleset still sends categories",
			config: AiBotsConfig{Active: types.BoolValue(false)},
			want: client.ManagedRule{RuleGroups: map[string]client.CoreRuleSet{
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
	for _, config := range []AiBotsConfig{
		{Active: types.BoolValue(true)},
		{Active: types.BoolValue(false)},
		{Active: types.BoolNull()},
		{Active: types.BoolValue(true), AiCrawlers: &CRSRuleConfig{Action: types.StringValue("deny")}},
		{Active: types.BoolValue(true), AiAssistants: &CRSRuleConfig{Action: types.StringValue("log")}},
		{Active: types.BoolValue(true), AiCrawlers: &CRSRuleConfig{Active: types.BoolValue(true), Action: types.StringValue("deny")}},
		{Active: types.BoolValue(true), AiAssistants: &CRSRuleConfig{Active: types.BoolValue(false), Action: types.StringValue("log")}},
		{Active: types.BoolValue(true), AiCrawlers: &CRSRuleConfig{Active: types.BoolValue(true), Action: types.StringValue("deny")}, AiAssistants: &CRSRuleConfig{Active: types.BoolValue(true), Action: types.StringValue("log")}},
		{Active: types.BoolValue(true), Action: types.StringValue("deny")},
	} {
		cfg := FirewallConfig{ManagedRulesets: &FirewallManagedRulesets{AiBots: &config}}
		api, err := cfg.toClient()
		if err != nil {
			t.Fatal(err)
		}
		got, err := fromClient(api, cfg, preserveConfiguredShape)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got.ManagedRulesets.AiBots, &config) {
			t.Errorf("configured shape changed: got %+v, want %+v", got.ManagedRulesets.AiBots, config)
		}
		roundTrip, err := got.toClient()
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(roundTrip.ManagedRulesets, api.ManagedRulesets) {
			t.Errorf("API round trip changed: got %+v, want %+v", roundTrip.ManagedRulesets, api.ManagedRulesets)
		}
		state, _ := firewallConfigStateAndPlan(t, &firewallConfigResource{}, got, got)
		var decoded FirewallConfig
		if diags := state.Get(context.Background(), &decoded); diags.HasError() {
			t.Fatal(diags)
		}
		if !reflect.DeepEqual(decoded.ManagedRulesets, got.ManagedRulesets) {
			t.Error("categories changed during Terraform state encoding")
		}
	}
}

func TestAiBotsImportCategories(t *testing.T) {
	rule := client.ManagedRule{Active: true, Action: "deny", RuleGroups: map[string]client.CoreRuleSet{
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
	for _, ref := range []*AiBotsConfig{
		{Active: types.BoolValue(true), AiCrawlers: &CRSRuleConfig{Action: types.StringValue("log")}},
		{Active: types.BoolValue(true), AiCrawlers: &CRSRuleConfig{Active: types.BoolValue(true), Action: types.StringValue("log")}},
		{Active: types.BoolNull()},
		{Active: types.BoolValue(true), Action: types.StringValue("log")},
	} {
		got := fromAiBots(client.ManagedRule{Active: true, RuleGroups: map[string]client.CoreRuleSet{"ai_crawlers": {Active: false, Action: "deny"}}}, ref, preserveConfiguredShape)
		if got.AiCrawlers == nil || got.AiCrawlers.Active.ValueBool() || got.AiCrawlers.Action.ValueString() != "deny" {
			t.Fatalf("category drift not reflected: %+v", got.AiCrawlers)
		}
		if !got.Active.ValueBool() || !got.Action.IsNull() {
			t.Fatalf("ruleset drift not reflected: %+v", got)
		}
	}
}

func TestAiBotsCategorySchema(t *testing.T) {
	ctx := context.Background()
	resp := &resource.SchemaResponse{}
	newFirewallConfigResource().Schema(ctx, resource.SchemaRequest{}, resp)
	bots := resp.Schema.Blocks["managed_rulesets"].(schema.SingleNestedBlock).Blocks["ai_bots"].(schema.SingleNestedBlock)
	for _, name := range []string{"ai_crawlers", "ai_assistants"} {
		category := bots.Attributes[name].(schema.SingleNestedAttribute)
		if !category.Optional || category.Computed || len(category.PlanModifiers) != 0 {
			t.Fatalf("%s should be optional without computed values or plan modifiers", name)
		}
		active := category.Attributes["active"].(schema.BoolAttribute)
		action := category.Attributes["action"].(schema.StringAttribute)
		if !active.Optional || active.Computed || active.Default != nil || !action.Required || action.Computed || action.Default != nil {
			t.Fatalf("%s should have an optional active and a required action, like OWASP", name)
		}
		for _, value := range []string{"log", "deny", "challenge", "allow", "invalid"} {
			validation := &validator.StringResponse{}
			action.Validators[0].ValidateString(ctx, validator.StringRequest{ConfigValue: types.StringValue(value)}, validation)
			if validation.Diagnostics.HasError() != (value != "log" && value != "deny") {
				t.Errorf("unexpected validation for %q: %v", value, validation.Diagnostics)
			}
		}
		cfg := FirewallConfig{ManagedRulesets: &FirewallManagedRulesets{AiBots: &AiBotsConfig{Action: types.StringValue("deny")}}}
		if name == "ai_crawlers" {
			cfg.ManagedRulesets.AiBots.AiCrawlers = &CRSRuleConfig{}
		} else {
			cfg.ManagedRulesets.AiBots.AiAssistants = &CRSRuleConfig{}
		}
		state, _ := firewallConfigStateAndPlan(t, &firewallConfigResource{}, cfg, cfg)
		validation := &validator.StringResponse{}
		bots.Attributes["action"].(schema.StringAttribute).Validators[0].ValidateString(ctx, validator.StringRequest{
			Path:        path.Root("managed_rulesets").AtName("ai_bots").AtName("action"),
			ConfigValue: cfg.ManagedRulesets.AiBots.Action,
			Config:      tfsdk.Config{Raw: state.Raw, Schema: state.Schema},
		}, validation)
		if !validation.Diagnostics.HasError() {
			t.Errorf("legacy action should conflict with %s", name)
		}
	}
}
