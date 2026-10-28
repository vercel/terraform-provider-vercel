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

func TestFirewallConfigAiBotsRequiresCrawlerAction(t *testing.T) {
	errors := validateFirewallConfigAiBots(t, AiBotsConfig{AiCrawlers: &RuleGroup{}})
	wantPath := tftypes.NewAttributePath().WithAttributeName("managed_rulesets").WithAttributeName("ai_bots").WithAttributeName("ai_crawlers").WithAttributeName("action")
	if len(errors) != 1 || !reflect.DeepEqual(errors[0].Attribute, wantPath) {
		t.Fatalf("expected a missing crawler action error at %s, got %v", wantPath, errors)
	}
}

func TestFirewallConfigAiBotsRequiresAssistantAction(t *testing.T) {
	errors := validateFirewallConfigAiBots(t, AiBotsConfig{AiAssistants: &RuleGroup{}})
	wantPath := tftypes.NewAttributePath().WithAttributeName("managed_rulesets").WithAttributeName("ai_bots").WithAttributeName("ai_assistants").WithAttributeName("action")
	if len(errors) != 1 || !reflect.DeepEqual(errors[0].Attribute, wantPath) {
		t.Fatalf("expected a missing assistant action error at %s, got %v", wantPath, errors)
	}
}

func TestFirewallConfigAiBotsRejectsCrawlerChallenge(t *testing.T) {
	errors := validateFirewallConfigAiBots(t, AiBotsConfig{
		AiCrawlers: &RuleGroup{Action: types.StringValue("challenge")},
	})
	wantPath := tftypes.NewAttributePath().WithAttributeName("managed_rulesets").WithAttributeName("ai_bots").WithAttributeName("ai_crawlers").WithAttributeName("action")
	if len(errors) != 1 || !reflect.DeepEqual(errors[0].Attribute, wantPath) {
		t.Fatalf("expected an invalid crawler action error at %s, got %v", wantPath, errors)
	}
}

func TestFirewallConfigAiBotsRejectsAssistantAllow(t *testing.T) {
	errors := validateFirewallConfigAiBots(t, AiBotsConfig{
		AiAssistants: &RuleGroup{Action: types.StringValue("allow")},
	})
	wantPath := tftypes.NewAttributePath().WithAttributeName("managed_rulesets").WithAttributeName("ai_bots").WithAttributeName("ai_assistants").WithAttributeName("action")
	if len(errors) != 1 || !reflect.DeepEqual(errors[0].Attribute, wantPath) {
		t.Fatalf("expected an invalid assistant action error at %s, got %v", wantPath, errors)
	}
}

func TestFirewallConfigAiBotsRejectsLegacyActionWithCrawlers(t *testing.T) {
	errors := validateFirewallConfigAiBots(t, AiBotsConfig{
		Action:     types.StringValue("deny"),
		AiCrawlers: &RuleGroup{Action: types.StringValue("log")},
	})
	wantPath := tftypes.NewAttributePath().WithAttributeName("managed_rulesets").WithAttributeName("ai_bots").WithAttributeName("action")
	if len(errors) != 1 || !reflect.DeepEqual(errors[0].Attribute, wantPath) {
		t.Fatalf("expected a legacy action conflict at %s, got %v", wantPath, errors)
	}
}

func TestFirewallConfigAiBotsRejectsLegacyActionWithAssistants(t *testing.T) {
	errors := validateFirewallConfigAiBots(t, AiBotsConfig{
		Action:       types.StringValue("deny"),
		AiAssistants: &RuleGroup{Action: types.StringValue("log")},
	})
	wantPath := tftypes.NewAttributePath().WithAttributeName("managed_rulesets").WithAttributeName("ai_bots").WithAttributeName("action")
	if len(errors) != 1 || !reflect.DeepEqual(errors[0].Attribute, wantPath) {
		t.Fatalf("expected a legacy action conflict at %s, got %v", wantPath, errors)
	}
}

func TestAiBotsToClientDefaults(t *testing.T) {
	config := AiBotsConfig{Active: types.BoolValue(true)}
	got := config.toClient()
	want := client.ManagedRule{Active: true, RuleGroups: map[string]client.RuleGroup{
		"ai_crawlers":   {Active: true, Action: "log"},
		"ai_assistants": {Active: true, Action: "log"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("AI Bots config:\ngot: %+v\nwant: %+v", got, want)
	}
}

func TestAiBotsToClientDefaultsOmittedAssistants(t *testing.T) {
	config := AiBotsConfig{
		Active:     types.BoolValue(true),
		AiCrawlers: &RuleGroup{Action: types.StringValue("deny")},
	}
	got := config.toClient()
	want := client.ManagedRule{Active: true, RuleGroups: map[string]client.RuleGroup{
		"ai_crawlers":   {Active: true, Action: "deny"},
		"ai_assistants": {Active: true, Action: "log"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("AI Bots config:\ngot: %+v\nwant: %+v", got, want)
	}
}

func TestAiBotsToClientDisabledAssistants(t *testing.T) {
	config := AiBotsConfig{
		Active:       types.BoolValue(true),
		AiAssistants: &RuleGroup{Active: types.BoolValue(false), Action: types.StringValue("deny")},
	}
	got := config.toClient()
	want := client.ManagedRule{Active: true, RuleGroups: map[string]client.RuleGroup{
		"ai_crawlers":   {Active: true, Action: "log"},
		"ai_assistants": {Active: false, Action: "deny"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("AI Bots config:\ngot: %+v\nwant: %+v", got, want)
	}
}

func TestAiBotsToClientDisabledRuleset(t *testing.T) {
	config := AiBotsConfig{Active: types.BoolValue(false)}
	got := config.toClient()
	want := client.ManagedRule{Active: false, RuleGroups: map[string]client.RuleGroup{
		"ai_crawlers":   {Active: true, Action: "log"},
		"ai_assistants": {Active: true, Action: "log"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("AI Bots config:\ngot: %+v\nwant: %+v", got, want)
	}
}

func TestAiBotsToClientLegacyAction(t *testing.T) {
	config := AiBotsConfig{Active: types.BoolValue(true), Action: types.StringValue("challenge")}
	got := config.toClient()
	want := client.ManagedRule{Active: true, Action: "challenge"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("AI Bots config:\ngot: %+v\nwant: %+v", got, want)
	}
}

func TestAiBotsReadPreservesOmittedValues(t *testing.T) {
	ref := &AiBotsConfig{AiCrawlers: &RuleGroup{Action: types.StringValue("deny")}}
	rule := client.ManagedRule{Active: false, RuleGroups: map[string]client.RuleGroup{
		"ai_crawlers":   {Active: true, Action: "deny"},
		"ai_assistants": {Active: true, Action: "log"},
	}}
	got := fromAiBots(rule, ref, preserveConfiguredShape)
	if !reflect.DeepEqual(got, ref) {
		t.Fatalf("omitted values changed:\ngot: %+v\nwant: %+v", got, ref)
	}
}

func TestAiBotsReadPreservesExplicitDefaults(t *testing.T) {
	ref := &AiBotsConfig{
		Active:     types.BoolValue(false),
		AiCrawlers: &RuleGroup{Active: types.BoolValue(true), Action: types.StringValue("log")},
	}
	rule := client.ManagedRule{Active: false, RuleGroups: map[string]client.RuleGroup{
		"ai_crawlers":   {Active: true, Action: "log"},
		"ai_assistants": {Active: true, Action: "log"},
	}}
	got := fromAiBots(rule, ref, preserveConfiguredShape)
	if !reflect.DeepEqual(got, ref) {
		t.Fatalf("explicit defaults changed:\ngot: %+v\nwant: %+v", got, ref)
	}
}

func TestAiBotsReadPreservesLegacyAction(t *testing.T) {
	ref := &AiBotsConfig{Active: types.BoolValue(true), Action: types.StringValue("deny")}
	got := fromAiBots(client.ManagedRule{Active: true, Action: "deny"}, ref, preserveConfiguredShape)
	if !reflect.DeepEqual(got, ref) {
		t.Fatalf("legacy config changed:\ngot: %+v\nwant: %+v", got, ref)
	}
}

func TestAiBotsReadDetectsLegacyMigration(t *testing.T) {
	ref := &AiBotsConfig{Active: types.BoolValue(true), Action: types.StringValue("log")}
	rule := client.ManagedRule{Active: true, RuleGroups: map[string]client.RuleGroup{
		"ai_crawlers":   {Active: true, Action: "deny"},
		"ai_assistants": {Active: true, Action: "log"},
	}}
	got := fromAiBots(rule, ref, preserveConfiguredShape)
	want := &AiBotsConfig{
		Active:     types.BoolValue(true),
		AiCrawlers: &RuleGroup{Active: types.BoolValue(true), Action: types.StringValue("deny")},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("legacy migration not reflected:\ngot: %+v\nwant: %+v", got, want)
	}
}

func TestAiBotsReadDetectsDrift(t *testing.T) {
	ref := &AiBotsConfig{AiCrawlers: &RuleGroup{Action: types.StringValue("log")}}
	rule := client.ManagedRule{Active: true, RuleGroups: map[string]client.RuleGroup{
		"ai_crawlers":   {Active: false, Action: "deny"},
		"ai_assistants": {Active: false, Action: "deny"},
	}}
	got := fromAiBots(rule, ref, preserveConfiguredShape)
	want := &AiBotsConfig{
		Active:       types.BoolValue(true),
		AiCrawlers:   &RuleGroup{Active: types.BoolValue(false), Action: types.StringValue("deny")},
		AiAssistants: &RuleGroup{Active: types.BoolValue(false), Action: types.StringValue("deny")},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("drift not reflected:\ngot: %+v\nwant: %+v", got, want)
	}
}

func TestAiBotsImportCategories(t *testing.T) {
	rule := client.ManagedRule{Active: true, Action: "deny", RuleGroups: map[string]client.RuleGroup{
		"ai_crawlers":   {Active: true, Action: "deny"},
		"ai_assistants": {Active: false, Action: "log"},
	}}
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

func validateFirewallConfigAiBots(t *testing.T, bots AiBotsConfig) []tfprotov6.Diagnostic {
	t.Helper()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	newFirewallConfigResource().Schema(ctx, resource.SchemaRequest{}, schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(ctx, FirewallConfig{
		ProjectID:       types.StringValue("prj_123"),
		ManagedRulesets: &FirewallManagedRulesets{AiBots: &bots},
	}); diags.HasError() {
		t.Fatal(diags)
	}
	config, err := tfprotov6.NewDynamicValue(state.Raw.Type(), state.Raw)
	if err != nil {
		t.Fatal(err)
	}
	server := providerserver.NewProtocol6(New())()
	response, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{
		TypeName: "vercel_firewall_config", Config: &config,
	})
	if err != nil {
		t.Fatal(err)
	}
	var errors []tfprotov6.Diagnostic
	for _, diag := range response.Diagnostics {
		if diag.Severity == tfprotov6.DiagnosticSeverityError {
			errors = append(errors, *diag)
		}
	}
	return errors
}
