package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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

func TestAlertRuleSchemaUsesV3Shape(t *testing.T) {
	rule := alertRuleSchema(t)
	for _, name := range []string{"type", "name", "rule_scope"} {
		if !rule.Attributes[name].IsRequired() {
			t.Fatalf("%s must be required", name)
		}
	}
	for _, name := range []string{"triggers", "match_minimum_severity_level", "severity", "evaluation", "trigger"} {
		if !rule.Attributes[name].IsOptional() {
			t.Fatalf("%s must be optional for the type-specific shape", name)
		}
	}
	if _, exists := rule.Attributes["notification_links"]; exists {
		t.Fatal("notification links must not be part of the alert rule resource")
	}
	if !rule.Attributes["is_default"].IsComputed() || !rule.Attributes["query_supported"].IsComputed() {
		t.Fatal("v3 compatibility metadata must be computed")
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

func TestAlertRuleCustomRoundTrip(t *testing.T) {
	ctx := context.Background()
	operator := "gt"
	threshold := 0.05
	minimum := 20.0
	querySupported := true
	rule, diags := alertRuleFromAPI(ctx, client.AlertRule{
		ID: "ar_custom", Type: client.AlertRuleTypeCustom, Name: "Error rate",
		RuleScope: client.AlertRuleScope{Type: "project", ProjectID: pointer("prj_123")},
		Severity:  pointer("medium"),
		Evaluation: &client.AlertRuleEvaluation{Window: "1h", Query: client.AlertRuleCustomQuery{
			Metrics: map[string]client.AlertRuleMetricSelection{
				"errors":   {Metric: "vercel.request.count", Aggregation: "sum", Filter: pointer("httpStatus>=500")},
				"requests": {Metric: "vercel.request.count", Aggregation: "sum"},
			},
			Formulas: map[string]string{"formula": "errors / requests"}, Outputs: []string{"formula"},
		}},
		Trigger: &client.AlertRuleCustomTrigger{
			Type: "threshold", Output: "formula", Operator: &operator, Threshold: &threshold,
			Minimum: &client.AlertRuleTriggerMinimum{Output: "errors", Threshold: minimum},
		},
		NotificationSettings: client.AlertRuleNotificationSettings{EnableTeamOwnerNotifications: false},
		QuerySupported:       &querySupported,
	}, types.StringValue("team_123"))
	if diags.HasError() {
		t.Fatalf("alertRuleFromAPI() diagnostics = %v", diags)
	}
	if rule.Evaluation == nil || rule.Trigger == nil || rule.QuerySupported.ValueBool() != true {
		t.Fatalf("rule = %#v", rule)
	}
	if len(rule.Evaluation.Query.Metrics.Elements()) != 2 || rule.Trigger.Minimum == nil {
		t.Fatalf("rule = %#v", rule)
	}

	payload, diags := rule.toCreateRequest(ctx)
	if diags.HasError() {
		t.Fatalf("toCreateRequest() diagnostics = %v", diags)
	}
	if payload.Evaluation == nil || payload.Trigger == nil || payload.Trigger.Minimum == nil {
		t.Fatalf("payload = %#v", payload)
	}
	if payload.Evaluation.Query.Formulas["formula"] != "errors / requests" || *payload.Trigger.Threshold != threshold {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestAlertRuleMetricDimensionsRoundTrip(t *testing.T) {
	configured := &AlertRuleEvaluation{
		Window: types.StringValue("5m"),
		Query: &AlertRuleCustomQuery{
			GroupBy: types.ListNull(types.StringType),
			Metrics: types.MapValueMust(alertRuleMetricAttrType, map[string]attr.Value{
				"users": types.ObjectValueMust(alertRuleMetricAttrType.AttrTypes, map[string]attr.Value{
					"metric": types.StringValue("custom.users"), "aggregation": types.StringValue("unique"),
					"per": types.StringNull(), "normalize": types.StringNull(),
					"dimensions": types.SetValueMust(types.StringType, []attr.Value{types.StringValue("userId")}),
					"filter":     types.StringNull(),
				}),
			}),
			Formulas: types.MapNull(types.StringType),
			Outputs:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("users")}),
		},
	}
	converted, diags := alertRuleEvaluationToClient(context.Background(), configured)
	if diags.HasError() {
		t.Fatalf("alertRuleEvaluationToClient() diagnostics = %v", diags)
	}
	if got := converted.Query.Metrics["users"].Dimensions; len(got) != 1 || got[0] != "userId" {
		t.Fatalf("dimensions = %#v", got)
	}
}

func pointer[T any](value T) *T {
	return &value
}
