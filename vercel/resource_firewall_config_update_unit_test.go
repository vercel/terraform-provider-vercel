package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func firewallConfigUpdateFixture() FirewallConfig {
	return FirewallConfig{
		ID:        types.StringValue("team_123/prj_123"),
		ProjectID: types.StringValue("prj_123"),
		TeamID:    types.StringValue("team_123"),
		Enabled:   types.BoolValue(true),
		ManagedRulesets: &FirewallManagedRulesets{
			BotProtection: &BotProtectionConfig{Active: types.BoolValue(true), Action: types.StringValue("log")},
			OWASP:         &CRSRule{XSS: &CRSRuleConfig{Active: types.BoolValue(false), Action: types.StringValue("log")}},
		},
		Rules: &FirewallRules{Rules: []FirewallRule{testResourceFirewallRule("rule_existing", "Block scanner probes", "/wp-admin", "deny")}},
		IPRules: &IPRules{Rules: []IPRule{{
			ID: types.StringValue("ip_existing"), Hostname: types.StringValue("example.com"),
			IP: types.StringValue("192.0.2.1"), Action: types.StringValue("deny"), Notes: types.StringValue(""),
		}}},
	}
}

func firewallConfigStateAndPlan(t *testing.T, r *firewallConfigResource, state, plan FirewallConfig) (tfsdk.State, tfsdk.Plan) {
	t.Helper()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	tfState := tfsdk.State{Schema: schemaResp.Schema}
	if diags := tfState.Set(ctx, state); diags.HasError() {
		t.Fatalf("setting state: %v", diags)
	}
	tfPlan := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := tfPlan.Set(ctx, plan); diags.HasError() {
		t.Fatalf("setting plan: %v", diags)
	}
	return tfState, tfPlan
}

func TestFirewallConfigModifyPlanUnknownSettings(t *testing.T) {
	tests := []struct {
		name  string
		path  path.Path
		value any
	}{
		{"enabled", path.Root("enabled"), types.BoolUnknown()},
		{"managed_active", path.Root("managed_rulesets").AtName("bot_protection").AtName("active"), types.BoolUnknown()},
		{"crs_active", path.Root("managed_rulesets").AtName("owasp").AtName("xss").AtName("active"), types.BoolUnknown()},
		{"ip_notes", path.Root("ip_rules").AtName("rule").AtListIndex(0).AtName("notes"), types.StringUnknown()},
		{"managed_block", path.Root("managed_rulesets"), nil},
		{"ip_block", path.Root("ip_rules"), nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			state, plan := firewallConfigUpdateFixture(), firewallConfigUpdateFixture()
			state.Enabled, plan.Enabled = types.BoolValue(false), types.BoolValue(false)
			state.ManagedRulesets.BotProtection.Active, plan.ManagedRulesets.BotProtection.Active = types.BoolValue(false), types.BoolValue(false)
			plan.Rules.Rules[0].ID = types.StringUnknown()
			r := &firewallConfigResource{}
			tfState, tfPlan := firewallConfigStateAndPlan(t, r, state, plan)
			if tc.value == nil {
				var attrs map[string]tftypes.Value
				if err := tfPlan.Raw.As(&attrs); err != nil {
					t.Fatal(err)
				}
				name := tc.path.String()
				attrs[name] = tftypes.NewValue(attrs[name].Type(), tftypes.UnknownValue)
				tfPlan.Raw = tftypes.NewValue(tfPlan.Raw.Type(), attrs)
			} else if diags := tfPlan.SetAttribute(ctx, tc.path, tc.value); diags.HasError() {
				t.Fatal(diags)
			}
			resp := &resource.ModifyPlanResponse{Plan: tfPlan}
			r.ModifyPlan(ctx, resource.ModifyPlanRequest{State: tfState, Plan: tfPlan}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("ModifyPlan with unknown settings: %v", resp.Diagnostics)
			}
			var id types.String
			if diags := resp.Plan.GetAttribute(ctx, path.Root("rules").AtName("rule").AtListIndex(0).AtName("id"), &id); diags.HasError() {
				t.Fatal(diags)
			}
			if !id.IsUnknown() {
				t.Errorf("rule ID = %s, want unknown while update method is undetermined", id)
			}
		})
	}
}

func TestFirewallConfigModifyPlanKnownRuleIDs(t *testing.T) {
	for _, scenario := range []string{"settings_changed", "settings_unknown", "unchanged"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			state, plan := firewallConfigUpdateFixture(), firewallConfigUpdateFixture()
			switch scenario {
			case "settings_changed":
				plan.Enabled = types.BoolValue(false)
			case "settings_unknown":
				plan.Enabled = types.BoolUnknown()
			}
			r := &firewallConfigResource{}
			tfState, tfPlan := firewallConfigStateAndPlan(t, r, state, plan)
			resp := &resource.ModifyPlanResponse{Plan: tfPlan}
			r.ModifyPlan(ctx, resource.ModifyPlanRequest{State: tfState, Plan: tfPlan}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			var id, ipID types.String
			if diags := resp.Plan.GetAttribute(ctx, path.Root("rules").AtName("rule").AtListIndex(0).AtName("id"), &id); diags.HasError() {
				t.Fatal(diags)
			}
			if diags := resp.Plan.GetAttribute(ctx, path.Root("ip_rules").AtName("rule").AtListIndex(0).AtName("id"), &ipID); diags.HasError() {
				t.Fatal(diags)
			}
			if scenario == "unchanged" {
				if !resp.Plan.Raw.Equal(tfState.Raw) {
					t.Error("unchanged plan must remain a no-op")
				}
			} else if !id.IsUnknown() {
				t.Errorf("custom-rule ID = %s, want unknown", id)
			}
			if !ipID.Equal(state.IPRules.Rules[0].ID) {
				t.Errorf("IP-rule ID changed from %s to %s", state.IPRules.Rules[0].ID, ipID)
			}
		})
	}
}

func TestFirewallConfigModifyPlanPreservesPATCHIdentityWhenUnknownsResolve(t *testing.T) {
	ctx := context.Background()
	state, plan := firewallConfigUpdateFixture(), firewallConfigUpdateFixture()
	state.Rules.Rules = []FirewallRule{
		testResourceFirewallRule("rule_a", "alpha", "/a", "deny"),
		testResourceFirewallRule("rule_b", "beta", "/b", "deny"),
	}
	plan.Rules.Rules = []FirewallRule{
		testResourceFirewallRule("", "", "/a", "deny"),
		testResourceFirewallRule("", "alpha", "/edited", "deny"),
	}
	plan.Rules.Rules[0].Name = types.StringUnknown()
	for i := range plan.Rules.Rules {
		plan.Rules.Rules[i].ID = types.StringUnknown()
	}
	r := &firewallConfigResource{}
	tfState, tfPlan := firewallConfigStateAndPlan(t, r, state, plan)
	initial := &resource.ModifyPlanResponse{Plan: tfPlan}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{State: tfState, Plan: tfPlan}, initial)
	if initial.Diagnostics.HasError() {
		t.Fatal(initial.Diagnostics)
	}
	if diags := initial.Plan.Get(ctx, &plan); diags.HasError() {
		t.Fatal(diags)
	}
	if plan.Rules.Rules[0].ID.ValueString() != "rule_b" || plan.Rules.Rules[1].ID.ValueString() != "rule_a" {
		t.Fatalf("initial correlation = [%s, %s], want [rule_b, rule_a]", plan.Rules.Rules[0].ID, plan.Rules.Rules[1].ID)
	}
	resolved := initial.Plan
	if diags := resolved.SetAttribute(ctx, path.Root("rules").AtName("rule").AtListIndex(0).AtName("name"), types.StringValue("alpha")); diags.HasError() {
		t.Fatal(diags)
	}
	final := &resource.ModifyPlanResponse{Plan: resolved}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{State: tfState, Plan: resolved}, final)
	if final.Diagnostics.HasError() {
		t.Fatal(final.Diagnostics)
	}
	var got FirewallConfig
	if diags := final.Plan.Get(ctx, &got); diags.HasError() {
		t.Fatal(diags)
	}
	for i, rule := range plan.Rules.Rules {
		if !got.Rules.Rules[i].ID.Equal(rule.ID) {
			t.Errorf("known PATCH ID changed when name resolved: rule[%d] was %s, now %s", i, rule.ID, got.Rules.Rules[i].ID)
		}
	}
}
