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
