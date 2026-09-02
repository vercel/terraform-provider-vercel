package vercel

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestFirewallConditionNegDefaultsToFalse(t *testing.T) {
	res := newFirewallConfigResource()

	resp := &resource.SchemaResponse{}
	res.Schema(context.Background(), resource.SchemaRequest{}, resp)

	rules := resp.Schema.Blocks["rules"].(schema.SingleNestedBlock)
	rule := rules.Blocks["rule"].(schema.ListNestedBlock)
	conditionGroups := rule.NestedObject.Attributes["condition_group"].(schema.ListNestedAttribute)
	conditions := conditionGroups.NestedObject.Attributes["conditions"].(schema.ListNestedAttribute)
	neg := conditions.NestedObject.Attributes["neg"].(schema.BoolAttribute)

	if neg.Default == nil {
		t.Fatal("neg should have a default")
	}
	if !neg.Computed {
		t.Fatal("neg should be computed so API values can be stored in state")
	}

	defaultResp := &defaults.BoolResponse{}
	neg.Default.DefaultBool(context.Background(), defaults.BoolRequest{}, defaultResp)
	if defaultResp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics applying neg default: %v", defaultResp.Diagnostics)
	}
	if defaultResp.PlanValue.IsNull() || defaultResp.PlanValue.ValueBool() {
		t.Fatalf("neg should default to false, got %s", defaultResp.PlanValue)
	}
}

func TestFromConditionPreservesNegFromAPI(t *testing.T) {
	for _, apiNeg := range []bool{false, true} {
		t.Run(fmt.Sprintf("neg_%t", apiNeg), func(t *testing.T) {
			condition, err := fromCondition(client.Condition{
				Type:  "header",
				Op:    "eq",
				Neg:   apiNeg,
				Key:   "x-origin-verify",
				Value: "secret",
			}, Condition{
				Neg: types.BoolNull(),
				Key: types.StringValue("x-origin-verify"),
			}, preserveConfiguredShape)
			if err != nil {
				t.Fatalf("unexpected error converting condition: %v", err)
			}
			if condition.Neg.IsNull() || condition.Neg.ValueBool() != apiNeg {
				t.Fatalf("expected API neg value %t, got %s", apiNeg, condition.Neg)
			}
		})
	}
}

func TestFirewallConfigResourceSchemaIncludesSessionFixationRule(t *testing.T) {
	res := newFirewallConfigResource()

	resp := &resource.SchemaResponse{}
	res.Schema(context.Background(), resource.SchemaRequest{}, resp)

	managedRulesets, ok := resp.Schema.Blocks["managed_rulesets"].(schema.SingleNestedBlock)
	if !ok {
		t.Fatalf("managed_rulesets block has unexpected type: %T", resp.Schema.Blocks["managed_rulesets"])
	}

	owasp, ok := managedRulesets.Blocks["owasp"].(schema.SingleNestedBlock)
	if !ok {
		t.Fatalf("owasp block has unexpected type: %T", managedRulesets.Blocks["owasp"])
	}

	sf, ok := owasp.Attributes["sf"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("sf attribute has unexpected type: %T", owasp.Attributes["sf"])
	}

	if !sf.Optional {
		t.Fatalf("sf should be optional")
	}

	action, ok := sf.Attributes["action"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("sf.action has unexpected type: %T", sf.Attributes["action"])
	}

	if !action.Required {
		t.Fatalf("sf.action should be required")
	}

	active, ok := sf.Attributes["active"].(schema.BoolAttribute)
	if !ok {
		t.Fatalf("sf.active has unexpected type: %T", sf.Attributes["active"])
	}

	if !active.Optional {
		t.Fatalf("sf.active should be optional")
	}
}

func TestFirewallConfigUsesModifyPlanForRuleIdentityCorrelation(t *testing.T) {
	res := newFirewallConfigResource()
	if _, ok := res.(resource.ResourceWithModifyPlan); !ok {
		t.Fatal("firewall config must implement ResourceWithModifyPlan to preserve rule IDs before apply")
	}
}

func TestFirewallConfigToClientIncludesSessionFixationRule(t *testing.T) {
	cfg := FirewallConfig{
		ProjectID: types.StringValue("prj_123"),
		TeamID:    types.StringValue("team_123"),
		Enabled:   types.BoolValue(true),
		ManagedRulesets: &FirewallManagedRulesets{
			OWASP: &CRSRule{
				SF: &CRSRuleConfig{
					Active: types.BoolValue(false),
					Action: types.StringValue("deny"),
				},
			},
		},
	}

	clientCfg, err := cfg.toClient()
	if err != nil {
		t.Fatalf("unexpected error converting config to client: %v", err)
	}

	if _, ok := clientCfg.ManagedRulesets["owasp"]; !ok {
		t.Fatalf("expected owasp managed ruleset to be set")
	}

	sf, ok := clientCfg.CRS["sf"]
	if !ok {
		t.Fatalf("expected sf key to be set in CRS map")
	}

	if sf.Action != "deny" {
		t.Fatalf("expected sf action to be deny, got %q", sf.Action)
	}

	if sf.Active {
		t.Fatalf("expected sf active to be false")
	}
}

func TestFromCRSIncludesSessionFixationRule(t *testing.T) {
	crsRules := defaultCRSMap()
	crsRules["sf"] = client.CoreRuleSet{
		Action: "deny",
		Active: false,
	}

	crs := fromCRS(crsRules, &FirewallManagedRulesets{
		OWASP: &CRSRule{
			SF: &CRSRuleConfig{
				Active: types.BoolValue(false),
				Action: types.StringValue("deny"),
			},
		},
	}, preserveConfiguredShape)

	if crs == nil {
		t.Fatalf("expected CRS value")
	}

	if crs.SF == nil {
		t.Fatalf("expected sf rule to be set")
	}

	if crs.SF.Action.ValueString() != "deny" {
		t.Fatalf("expected sf action to be deny, got %q", crs.SF.Action.ValueString())
	}

	if crs.SF.Active.ValueBool() {
		t.Fatalf("expected sf active to be false")
	}
}

func importFirewallConfig(t *testing.T, apiConfig client.FirewallConfig) FirewallConfig {
	t.Helper()
	got, err := fromClient(apiConfig, FirewallConfig{
		ProjectID: types.StringValue(apiConfig.ProjectID),
		TeamID:    types.StringValue(apiConfig.TeamID),
	}, canonicalImportShape)
	if err != nil {
		t.Fatalf("unexpected error converting imported config: %v", err)
	}
	return got
}

func TestCanonicalImportEnabled(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("enabled_%t", enabled), func(t *testing.T) {
			got := importFirewallConfig(t, client.FirewallConfig{ProjectID: "prj_123", TeamID: "team_123", Enabled: enabled})
			if got.Enabled.IsNull() || got.Enabled.ValueBool() != enabled {
				t.Fatalf("enabled = %s, want %t", got.Enabled, enabled)
			}
		})
	}
}

func TestCanonicalImportRuleOptionalStringsAndActive(t *testing.T) {
	tests := []struct {
		name, description, duration           string
		active                                bool
		wantDescriptionNull, wantDurationNull bool
	}{
		{name: "empty enabled", active: true, wantDescriptionNull: true, wantDurationNull: true},
		{name: "nonempty disabled", description: "description", duration: "1h", active: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := importFirewallConfig(t, client.FirewallConfig{Rules: []client.FirewallRule{{
				Active: tc.active, Description: tc.description,
				Action: client.Action{Mitigate: client.Mitigate{Action: "deny", ActionDuration: tc.duration}},
			}}})
			rule := got.Rules.Rules[0]
			if rule.Active.IsNull() || rule.Active.ValueBool() != tc.active {
				t.Fatalf("active = %s, want %t", rule.Active, tc.active)
			}
			if rule.Description.IsNull() != tc.wantDescriptionNull || !rule.Description.IsNull() && rule.Description.ValueString() != tc.description {
				t.Fatalf("description = %s", rule.Description)
			}
			if rule.Action.ActionDuration.IsNull() != tc.wantDurationNull || !rule.Action.ActionDuration.IsNull() && rule.Action.ActionDuration.ValueString() != tc.duration {
				t.Fatalf("action_duration = %s", rule.Action.ActionDuration)
			}
		})
	}
}

func TestCanonicalImportConditions(t *testing.T) {
	tests := []struct {
		name                       string
		condition                  client.Condition
		wantKeyNull, wantValueNull bool
		wantValue                  string
		wantValues                 []string
	}{
		{name: "keyed scalar", condition: client.Condition{Type: "query", Op: "eq", Key: "campaign", Value: "spring"}, wantValue: "spring"},
		{name: "keyed empty scalar", condition: client.Condition{Type: "query", Op: "eq", Key: "campaign", Value: ""}, wantValue: ""},
		{name: "keyless scalar", condition: client.Condition{Type: "path", Op: "eq", Value: "/sale"}, wantKeyNull: true, wantValue: "/sale"},
		{name: "keyed list", condition: client.Condition{Type: "query", Op: "inc", Key: "campaign", Value: []any{"spring", "summer"}}, wantValues: []string{"spring", "summer"}, wantValueNull: true},
		{name: "keyless list", condition: client.Condition{Type: "path", Op: "inc", Value: []any{"/a", "/b"}}, wantKeyNull: true, wantValues: []string{"/a", "/b"}, wantValueNull: true},
		{name: "keyed existence", condition: client.Condition{Type: "header", Op: "ex", Key: "x-test", Value: ""}, wantValueNull: true},
		{name: "keyless existence", condition: client.Condition{Type: "path", Op: "nex", Value: nil}, wantKeyNull: true, wantValueNull: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := fromCondition(tc.condition, Condition{}, canonicalImportShape)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Key.IsNull() != tc.wantKeyNull {
				t.Fatalf("key = %s", got.Key)
			}
			if !got.Key.IsNull() && got.Key.ValueString() != tc.condition.Key {
				t.Fatalf("key = %q", got.Key.ValueString())
			}
			if got.Value.IsNull() != tc.wantValueNull {
				t.Fatalf("value = %s", got.Value)
			}
			if !got.Value.IsNull() && got.Value.ValueString() != tc.wantValue {
				t.Fatalf("value = %q", got.Value.ValueString())
			}
			if tc.wantValues != nil {
				var values []string
				diags := got.Values.ElementsAs(context.Background(), &values, false)
				if diags.HasError() || fmt.Sprint(values) != fmt.Sprint(tc.wantValues) {
					t.Fatalf("values = %v, diagnostics = %v", values, diags)
				}
			}
		})
	}
}

func TestCanonicalImportManagedRulesets(t *testing.T) {
	got := importFirewallConfig(t, client.FirewallConfig{ManagedRulesets: map[string]client.ManagedRule{
		"bot_protection": {Active: true, Action: "challenge"},
		"ai_bots":        {Active: false, Action: "deny"},
		"future_rule":    {Active: true, Action: "deny"},
	}})
	if got.ManagedRulesets == nil || got.ManagedRulesets.BotProtection == nil || got.ManagedRulesets.AiBots == nil {
		t.Fatalf("recognized managed rules missing: %+v", got.ManagedRulesets)
	}
	if got.ManagedRulesets.BotFilter != nil {
		t.Fatal("import invented deprecated bot_filter")
	}
	if !got.ManagedRulesets.BotProtection.Active.ValueBool() || got.ManagedRulesets.BotProtection.Action.ValueString() != "challenge" {
		t.Fatalf("bot protection = %+v", got.ManagedRulesets.BotProtection)
	}
	if got.ManagedRulesets.AiBots.Active.ValueBool() || got.ManagedRulesets.AiBots.Action.ValueString() != "deny" {
		t.Fatalf("ai bots = %+v", got.ManagedRulesets.AiBots)
	}

	unknownOnly := importFirewallConfig(t, client.FirewallConfig{ManagedRulesets: map[string]client.ManagedRule{"future_rule": {Active: true, Action: "deny"}}})
	if unknownOnly.ManagedRulesets != nil {
		t.Fatalf("unknown managed rule materialized state: %+v", unknownOnly.ManagedRulesets)
	}
}

func TestCanonicalImportCRS(t *testing.T) {
	tests := []struct {
		name                       string
		managed                    map[string]client.ManagedRule
		crs                        map[string]client.CoreRuleSet
		wantOWASP, wantSF, wantXSS bool
	}{
		{name: "default only", crs: defaultCRSMap()},
		{name: "partial", crs: map[string]client.CoreRuleSet{"sf": {Active: false, Action: "deny"}}, wantOWASP: true, wantSF: true},
		{name: "partial active", crs: map[string]client.CoreRuleSet{"xss": {Active: true, Action: "log"}}, wantOWASP: true, wantXSS: true},
		{name: "unknown only", crs: map[string]client.CoreRuleSet{"future": {Active: true, Action: "deny"}}},
		{name: "active marker without details", managed: map[string]client.ManagedRule{"owasp": {Active: true}}, wantOWASP: true},
		{name: "active marker with defaults", managed: map[string]client.ManagedRule{"owasp": {Active: true}}, crs: defaultCRSMap(), wantOWASP: true},
		{name: "active marker with unknown category", managed: map[string]client.ManagedRule{"owasp": {Active: true}}, crs: map[string]client.CoreRuleSet{"future": {Active: true, Action: "deny"}}, wantOWASP: true},
		{name: "active marker with partial category", managed: map[string]client.ManagedRule{"owasp": {Active: true}}, crs: map[string]client.CoreRuleSet{"sf": {Active: false, Action: "deny"}}, wantOWASP: true, wantSF: true},
		{name: "inactive marker overrides category", managed: map[string]client.ManagedRule{"owasp": {Active: false}}, crs: map[string]client.CoreRuleSet{"sf": {Active: false, Action: "deny"}}},
		{name: "crs only response", crs: map[string]client.CoreRuleSet{"sf": {Active: false, Action: "deny"}}, wantOWASP: true, wantSF: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := importFirewallConfig(t, client.FirewallConfig{ManagedRulesets: tc.managed, CRS: tc.crs})
			owasp := (*CRSRule)(nil)
			if got.ManagedRulesets != nil {
				owasp = got.ManagedRulesets.OWASP
			}
			if (owasp != nil) != tc.wantOWASP {
				t.Fatalf("OWASP = %+v", owasp)
			}
			if owasp != nil && (owasp.SF != nil) != tc.wantSF {
				t.Fatalf("SF = %+v", owasp.SF)
			}
			if owasp != nil && (owasp.XSS != nil) != tc.wantXSS {
				t.Fatalf("XSS = %+v", owasp.XSS)
			}
			if owasp != nil && owasp.SF != nil && (owasp.SF.Active.IsNull() || owasp.SF.Action.ValueString() != "deny") {
				t.Fatalf("SF = %+v", owasp.SF)
			}
			roundTrip, err := got.toClient()
			if err != nil {
				t.Fatalf("round trip failed: %v", err)
			}
			_, hasOWASPMarker := roundTrip.ManagedRulesets["owasp"]
			if hasOWASPMarker != tc.wantOWASP {
				t.Fatalf("round-trip OWASP marker = %t", hasOWASPMarker)
			}
		})
	}
}

func TestCanonicalImportIPRuleNotes(t *testing.T) {
	got := importFirewallConfig(t, client.FirewallConfig{IPRules: []client.IPRule{
		{ID: "empty", Notes: "", Action: "deny"},
		{ID: "noted", Notes: "keep me", Action: "deny"},
	}})
	if !got.IPRules.Rules[0].Notes.IsNull() {
		t.Fatalf("empty notes = %s", got.IPRules.Rules[0].Notes)
	}
	if got.IPRules.Rules[1].Notes.IsNull() || got.IPRules.Rules[1].Notes.ValueString() != "keep me" {
		t.Fatalf("nonempty notes = %s", got.IPRules.Rules[1].Notes)
	}
}

func TestPreserveConfiguredShapeMode(t *testing.T) {
	apiConfig := client.FirewallConfig{
		Enabled:         true,
		Rules:           []client.FirewallRule{{Active: true, Description: "", Action: client.Action{Mitigate: client.Mitigate{Action: "deny"}}, ConditionGroup: []client.ConditionGroup{{Conditions: []client.Condition{{Type: "query", Op: "eq", Key: "api-key", Value: "api-value"}}}}}},
		IPRules:         []client.IPRule{{Notes: "", Action: "deny"}},
		ManagedRulesets: map[string]client.ManagedRule{"bot_protection": {Active: true, Action: "challenge"}, "ai_bots": {Active: true, Action: "deny"}},
	}
	state := FirewallConfig{
		Enabled:         types.BoolNull(),
		Rules:           &FirewallRules{Rules: []FirewallRule{{Active: types.BoolNull(), Description: types.StringNull(), Action: Mitigate{ActionDuration: types.StringNull()}, ConditionGroup: []ConditionGroup{{Conditions: []Condition{{Key: types.StringNull(), Value: types.StringNull()}}}}}}},
		IPRules:         &IPRules{Rules: []IPRule{{Notes: types.StringNull()}}},
		ManagedRulesets: &FirewallManagedRulesets{BotFilter: &BotFilterConfig{}, AiBots: &AiBotsConfig{}},
	}
	got, err := fromClient(apiConfig, state, preserveConfiguredShape)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Enabled.IsNull() || !got.Rules.Rules[0].Active.IsNull() || !got.Rules.Rules[0].Description.IsNull() || !got.Rules.Rules[0].Action.ActionDuration.IsNull() {
		t.Fatalf("optional configured shape changed: %+v", got.Rules.Rules[0])
	}
	condition := got.Rules.Rules[0].ConditionGroup[0].Conditions[0]
	if !condition.Key.IsNull() || condition.Value.IsNull() || condition.Value.ValueString() != "api-value" {
		t.Fatalf("condition shape changed: %+v", condition)
	}
	if !got.IPRules.Rules[0].Notes.IsNull() {
		t.Fatalf("notes shape changed: %s", got.IPRules.Rules[0].Notes)
	}
	if got.ManagedRulesets.BotFilter == nil || got.ManagedRulesets.BotProtection != nil || got.ManagedRulesets.AiBots == nil {
		t.Fatalf("managed rules shape changed: %+v", got.ManagedRulesets)
	}
}

func defaultCRSMap() map[string]client.CoreRuleSet {
	return map[string]client.CoreRuleSet{
		"xss": {
			Action: "log",
			Active: false,
		},
		"sqli": {
			Action: "log",
			Active: false,
		},
		"sf": {
			Action: "log",
			Active: false,
		},
		"lfi": {
			Action: "log",
			Active: false,
		},
		"rfi": {
			Action: "log",
			Active: false,
		},
		"rce": {
			Action: "log",
			Active: false,
		},
		"sd": {
			Action: "log",
			Active: false,
		},
		"ma": {
			Action: "log",
			Active: false,
		},
		"php": {
			Action: "log",
			Active: false,
		},
		"gen": {
			Action: "log",
			Active: false,
		},
		"java": {
			Action: "log",
			Active: false,
		},
	}
}

func TestMatchFirewallRulesPreservesStableRuleIDs(t *testing.T) {
	current := []client.FirewallRule{
		testClientFirewallRule("rule_a", "alpha", "/alpha", "deny"),
		testClientFirewallRule("rule_b", "beta", "/beta", "deny"),
		testClientFirewallRule("rule_c", "charlie", "/charlie", "deny"),
	}
	desired := []client.FirewallRule{
		testClientFirewallRule("", "beta", "/beta", "deny"),
		testClientFirewallRule("", "alpha-renamed", "/alpha", "deny"),
		testClientFirewallRule("", "delta", "/delta", "deny"),
	}

	matches, removals, inserts, err := matchFirewallRules(current, desired)
	if err != nil {
		t.Fatalf("unexpected match error: %v", err)
	}

	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
	if len(removals) != 1 || removals[0] != 2 {
		t.Fatalf("expected rule_c removal, got %v", removals)
	}
	if len(inserts) != 1 || inserts[0] != 2 {
		t.Fatalf("expected delta insert, got %v", inserts)
	}

	gotMatches := map[int]int{}
	for _, match := range matches {
		gotMatches[match.currentIndex] = match.desiredIndex
	}

	if gotMatches[0] != 1 {
		t.Fatalf("expected rule_a to match renamed alpha rule, got %+v", matches)
	}
	if gotMatches[1] != 0 {
		t.Fatalf("expected rule_b to match beta rule, got %+v", matches)
	}
}

func TestMatchFirewallRulesMatchesByNameWhenRuleBodyChanges(t *testing.T) {
	current := []client.FirewallRule{
		testClientFirewallRule("rule_a", "alpha", "/alpha", "deny"),
	}
	desired := []client.FirewallRule{
		testClientFirewallRule("", "alpha", "/renamed", "challenge"),
	}

	matches, removals, inserts, err := matchFirewallRules(current, desired)
	if err != nil {
		t.Fatalf("unexpected match error: %v", err)
	}

	if len(matches) != 1 {
		t.Fatalf("expected one match, got %d", len(matches))
	}
	if matches[0].currentIndex != 0 || matches[0].desiredIndex != 0 {
		t.Fatalf("unexpected match mapping: %+v", matches)
	}
	if len(removals) != 0 {
		t.Fatalf("expected no removals, got %v", removals)
	}
	if len(inserts) != 0 {
		t.Fatalf("expected no inserts, got %v", inserts)
	}
}

func TestPropagateFirewallRuleIDsTreatsCompleteEditAsUpdate(t *testing.T) {
	state := &FirewallRules{Rules: []FirewallRule{
		testResourceFirewallRule("rule_a", "alpha", "/alpha", "deny"),
	}}
	plan := &FirewallRules{Rules: []FirewallRule{
		testResourceFirewallRule("", "renamed", "/changed", "challenge"),
	}}
	plan.Rules[0].ID = types.StringUnknown()

	if err := propagateFirewallRuleIDs(state, plan); err != nil {
		t.Fatalf("propagateFirewallRuleIDs() returned an error: %v", err)
	}
	if got := plan.Rules[0].ID.ValueString(); got != "rule_a" {
		t.Fatalf("planned rule ID = %q, want rule_a", got)
	}
}

func TestPropagateFirewallRuleIDsFollowsRulesAcrossReorderAndInsertion(t *testing.T) {
	state := &FirewallRules{Rules: []FirewallRule{
		testResourceFirewallRule("rule_a", "alpha", "/alpha", "deny"),
		testResourceFirewallRule("rule_b", "beta", "/beta", "deny"),
	}}
	plan := &FirewallRules{Rules: []FirewallRule{
		testResourceFirewallRule("", "new-rule", "/new", "deny"),
		testResourceFirewallRule("", "beta", "/beta", "deny"),
		testResourceFirewallRule("", "alpha", "/alpha", "deny"),
	}}
	for i := range plan.Rules {
		plan.Rules[i].ID = types.StringUnknown()
	}

	if err := propagateFirewallRuleIDs(state, plan); err != nil {
		t.Fatalf("propagateFirewallRuleIDs() returned an error: %v", err)
	}
	if !plan.Rules[0].ID.IsUnknown() {
		t.Fatalf("new planned rule ID = %v, want unknown", plan.Rules[0].ID)
	}
	if got := plan.Rules[1].ID.ValueString(); got != "rule_b" {
		t.Fatalf("beta planned rule ID = %q, want rule_b", got)
	}
	if got := plan.Rules[2].ID.ValueString(); got != "rule_a" {
		t.Fatalf("alpha planned rule ID = %q, want rule_a", got)
	}
}

func TestOnlyFirewallRulesChangedIgnoresIPRuleIDs(t *testing.T) {
	state := FirewallConfig{
		ProjectID: types.StringValue("prj_123"),
		TeamID:    types.StringValue("team_123"),
		Enabled:   types.BoolValue(true),
		IPRules: &IPRules{
			Rules: []IPRule{
				{
					ID:       types.StringValue("ip_123"),
					Hostname: types.StringValue("example.com"),
					IP:       types.StringValue("1.2.3.4"),
					Action:   types.StringValue("deny"),
					Notes:    types.StringNull(),
				},
			},
		},
		Rules: &FirewallRules{
			Rules: []FirewallRule{
				testResourceFirewallRule("rule_123", "alpha", "/alpha", "deny"),
			},
		},
	}
	plan := FirewallConfig{
		ProjectID: types.StringValue("prj_123"),
		TeamID:    types.StringValue("team_123"),
		Enabled:   types.BoolValue(true),
		IPRules: &IPRules{
			Rules: []IPRule{
				{
					ID:       types.StringNull(),
					Hostname: types.StringValue("example.com"),
					IP:       types.StringValue("1.2.3.4"),
					Action:   types.StringValue("deny"),
					Notes:    types.StringNull(),
				},
			},
		},
		Rules: &FirewallRules{
			Rules: []FirewallRule{
				testResourceFirewallRule("", "alpha-renamed", "/alpha", "deny"),
			},
		},
	}

	onlyRulesChanged, err := onlyFirewallRulesChanged(state, plan)
	if err != nil {
		t.Fatalf("unexpected compare error: %v", err)
	}
	if !onlyRulesChanged {
		t.Fatalf("expected onlyFirewallRulesChanged to ignore IP rule IDs")
	}
}

func TestMoveFirewallRuleID(t *testing.T) {
	ids := []string{"rule_a", "rule_b", "rule_c"}
	ids = moveFirewallRuleID(ids, 1, 0)

	if ids[0] != "rule_b" || ids[1] != "rule_a" || ids[2] != "rule_c" {
		t.Fatalf("unexpected rule order after move: %v", ids)
	}
}

func testClientFirewallRule(id, name, path, action string) client.FirewallRule {
	return client.FirewallRule{
		ID:          id,
		Name:        name,
		Description: "",
		Active:      true,
		ConditionGroup: []client.ConditionGroup{
			{
				Conditions: []client.Condition{
					{
						Type:  "path",
						Op:    "eq",
						Neg:   false,
						Key:   "",
						Value: path,
					},
				},
			},
		},
		Action: client.Action{
			Mitigate: client.Mitigate{
				Action:         action,
				ActionDuration: "",
			},
		},
	}
}

func testResourceFirewallRule(id, name, path, action string) FirewallRule {
	ruleID := types.StringNull()
	if id != "" {
		ruleID = types.StringValue(id)
	}

	return FirewallRule{
		ID:          ruleID,
		Name:        types.StringValue(name),
		Description: types.StringNull(),
		Active:      types.BoolValue(true),
		ConditionGroup: []ConditionGroup{
			{
				Conditions: []Condition{
					{
						Type:   types.StringValue("path"),
						Op:     types.StringValue("eq"),
						Neg:    types.BoolValue(false),
						Key:    types.StringNull(),
						Value:  types.StringValue(path),
						Values: types.ListNull(types.StringType),
					},
				},
			},
		},
		Action: Mitigate{
			Action:         types.StringValue(action),
			RateLimit:      types.ObjectNull(ratelimitType.AttrTypes),
			Redirect:       types.ObjectNull(redirectType.AttrTypes),
			ActionDuration: types.StringNull(),
		},
	}
}
