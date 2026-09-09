package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func alertRuleSchema(t *testing.T) schema.Schema {
	t.Helper()
	var response resource.SchemaResponse
	newAlertRuleResource().Schema(context.Background(), resource.SchemaRequest{}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics = %v", response.Diagnostics)
	}
	return response.Schema
}

func TestAlertRuleSchemaUsesBuiltInShape(t *testing.T) {
	rule := alertRuleSchema(t)
	for _, name := range []string{"type", "name", "rule_scope", "triggers", "match_minimum_severity_level"} {
		if !rule.Attributes[name].IsRequired() {
			t.Fatalf("%s must be required", name)
		}
	}
	if !rule.Attributes["notification_settings"].IsOptional() || !rule.Attributes["is_default"].IsComputed() {
		t.Fatal("notification settings must be optional and default metadata must be computed")
	}
}

func TestAlertRuleBuiltInRoundTrip(t *testing.T) {
	ctx := context.Background()
	filter := "statusGroup:5xx"
	severity := "high"
	rule, diags := alertRuleFromAPI(ctx, client.AlertRule{
		ID: "ar_123", Type: client.AlertRuleTypeBuiltIn, Name: "Errors",
		RuleScope:                 client.AlertRuleScope{Type: "include", ProjectIDs: []string{"prj_123"}},
		Triggers:                  &client.AlertRuleTriggers{Mode: "selected", Items: []client.AlertRuleTrigger{{Type: "error_anomaly", Filter: &filter}}},
		MatchMinimumSeverityLevel: &severity,
		NotificationSettings:      client.AlertRuleNotificationSettings{EnableTeamOwnerNotifications: true},
	}, types.StringValue("team_123"))
	if diags.HasError() {
		t.Fatalf("alertRuleFromAPI() diagnostics = %v", diags)
	}
	if rule.Type.ValueString() != client.AlertRuleTypeBuiltIn || rule.RuleScope.Type.ValueString() != "include" {
		t.Fatalf("rule = %#v", rule)
	}
	if len(rule.Triggers.Elements()) != 1 || len(rule.RuleScope.ProjectIDs.Elements()) != 1 {
		t.Fatalf("rule = %#v", rule)
	}

	payload, diags := rule.toCreateRequest(ctx)
	if diags.HasError() {
		t.Fatalf("toCreateRequest() diagnostics = %v", diags)
	}
	if payload.Triggers == nil || payload.Triggers.Mode != "selected" || payload.Triggers.Items[0].Filter == nil {
		t.Fatalf("payload = %#v", payload)
	}
	if payload.RuleScope.ProjectIDs[0] != "prj_123" || payload.NotificationSettings == nil || !payload.NotificationSettings.EnableTeamOwnerNotifications {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestAlertRuleAllTriggersAreExpandedForTerraformState(t *testing.T) {
	triggers, diags := alertRuleTriggersFromClient(context.Background(), &client.AlertRuleTriggers{Mode: "all"})
	if diags.HasError() {
		t.Fatalf("alertRuleTriggersFromClient() diagnostics = %v", diags)
	}
	if len(triggers.Elements()) != len(client.AlertRuleBuiltInTriggerTypes) {
		t.Fatalf("trigger count = %d, want %d", len(triggers.Elements()), len(client.AlertRuleBuiltInTriggerTypes))
	}
}
