package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

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
	})

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
