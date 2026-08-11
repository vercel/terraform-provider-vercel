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

func TestAlertRuleSchemaSeparatesProjectTargeting(t *testing.T) {
	alertRule := alertRuleSchema(t)

	if !alertRule.Attributes["alert_types"].IsRequired() {
		t.Fatalf("alert_types must be required")
	}
	// The API stores one projectId field, but it means different things for
	// built-in and custom rules, so the two are separate optional attributes.
	for _, name := range []string{"project_filter", "project_id"} {
		attribute := alertRule.Attributes[name]
		if !attribute.IsOptional() || attribute.IsRequired() {
			t.Fatalf("%s must be optional and not required", name)
		}
	}

	customAlert, ok := alertRule.Attributes["custom_alert"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("custom_alert schema = %T, want schema.SingleNestedAttribute", alertRule.Attributes["custom_alert"])
	}
	if !customAlert.IsOptional() {
		t.Fatalf("custom_alert must be optional")
	}
	for _, name := range []string{"event", "rollups", "trigger_type", "trigger_operator", "trigger_threshold"} {
		if !customAlert.Attributes[name].IsRequired() {
			t.Fatalf("custom_alert.%s must be required", name)
		}
	}
	// granularity is computed as well as optional because the API defaults it.
	granularity := customAlert.Attributes["granularity"]
	if !granularity.IsOptional() || !granularity.IsComputed() {
		t.Fatalf("custom_alert.granularity must be optional and computed")
	}
}

func TestAlertRuleGranularityRoundTrip(t *testing.T) {
	for _, granularity := range []string{"5m", "1h", "1d"} {
		converted, err := alertRuleGranularityToClient(types.StringValue(granularity))
		if err != nil {
			t.Fatalf("alertRuleGranularityToClient(%q) error = %v", granularity, err)
		}
		if got := alertRuleGranularityFromClient(converted).ValueString(); got != granularity {
			t.Fatalf("granularity round trip = %q, want %q", got, granularity)
		}
	}

	converted, err := alertRuleGranularityToClient(types.StringNull())
	if err != nil {
		t.Fatalf("alertRuleGranularityToClient(null) error = %v", err)
	}
	if converted != nil {
		t.Fatalf("alertRuleGranularityToClient(null) = %#v, want nil", converted)
	}
	if got := alertRuleGranularityFromClient(nil); !got.IsNull() {
		t.Fatalf("alertRuleGranularityFromClient(nil) = %#v, want null", got)
	}

	if _, err := alertRuleGranularityToClient(types.StringValue("1w")); err == nil {
		t.Fatalf("alertRuleGranularityToClient(\"1w\") error = nil, want an error")
	}
}

func TestAlertRuleFromAPIMapsProjectFilterForBuiltInRules(t *testing.T) {
	projectFilter := "projectId in ('prj_123')"
	sensitivity := int64(4)
	autosubscribe := true
	filter := "statusGroup eq '5xx'"

	result, diags := alertRuleFromAPI(context.Background(), client.AlertRule{
		ID:     "ar_123",
		TeamID: "team_123",
		Name:   "5xx anomalies",
		// The API returns the OData filter in the same field custom alerts use
		// for a raw project ID.
		ProjectID: &projectFilter,
		AlertTypes: []client.AlertRuleType{
			{Type: client.AlertRuleTypeErrorAnomaly, Filter: &filter},
		},
		SensitivityLevel:    &sensitivity,
		AutosubscribeOwners: &autosubscribe,
	})
	if diags.HasError() {
		t.Fatalf("alertRuleFromAPI() diagnostics = %v", diags)
	}

	if result.ProjectFilter.ValueString() != projectFilter {
		t.Fatalf("project_filter = %q, want %q", result.ProjectFilter.ValueString(), projectFilter)
	}
	if !result.ProjectID.IsNull() {
		t.Fatalf("project_id = %#v, want null for a built-in rule", result.ProjectID)
	}
	if result.CustomAlert != nil {
		t.Fatalf("custom_alert = %#v, want nil", result.CustomAlert)
	}
	if result.SensitivityLevel.ValueInt64() != 4 || !result.AutosubscribeOwners.ValueBool() {
		t.Fatalf("result = %#v", result)
	}
	// Fields the API omits stay null so that they round trip against a config
	// that does not set them.
	if !result.AutosubscribeProjectAdmins.IsNull() {
		t.Fatalf("autosubscribe_project_admins = %#v, want null", result.AutosubscribeProjectAdmins)
	}
	if result.AlertTypes.IsNull() || len(result.AlertTypes.Elements()) != 1 {
		t.Fatalf("alert_types = %#v", result.AlertTypes)
	}
}

func TestAlertRuleFromAPIMapsProjectIDForCustomAlertRules(t *testing.T) {
	projectID := "prj_123"
	minThreshold := 20.0
	hours := int64(1)
	rollupFilter := "httpStatus ge 500"

	result, diags := alertRuleFromAPI(context.Background(), client.AlertRule{
		ID:         "ar_custom",
		TeamID:     "team_123",
		Name:       "Checkout error rate",
		ProjectID:  &projectID,
		AlertTypes: []client.AlertRuleType{{Type: client.AlertRuleTypeCustomAlert}},
		CustomAlert: &client.CustomAlert{
			Query: client.CustomAlertQuery{
				// The scope is derived from the rule's team and project, so it
				// is not surfaced in state.
				Scope: &client.CustomAlertQueryScope{Type: "project", OwnerID: "team_123", ProjectIDs: []string{projectID}},
				Event: "incomingRequest",
				Rollups: map[string]client.CustomAlertRollup{
					"errors":   {Measure: "count", Aggregation: "sum", Filter: &rollupFilter},
					"requests": {Measure: "count", Aggregation: "sum"},
				},
				Granularity: &client.CustomAlertGranularity{Hours: &hours},
			},
			TriggerType:      "threshold",
			TriggerOperator:  "gt",
			TriggerThreshold: 0.05,
			MinThreshold:     &minThreshold,
			Formula:          &client.CustomAlertFormula{Operator: "divide", Left: "errors", Right: "requests"},
		},
	})
	if diags.HasError() {
		t.Fatalf("alertRuleFromAPI() diagnostics = %v", diags)
	}

	if result.ProjectID.ValueString() != projectID {
		t.Fatalf("project_id = %q, want %q", result.ProjectID.ValueString(), projectID)
	}
	if !result.ProjectFilter.IsNull() {
		t.Fatalf("project_filter = %#v, want null for a custom alert rule", result.ProjectFilter)
	}
	if result.CustomAlert == nil {
		t.Fatalf("custom_alert = nil")
	}
	if result.CustomAlert.Granularity.ValueString() != "1h" {
		t.Fatalf("granularity = %q, want 1h", result.CustomAlert.Granularity.ValueString())
	}
	if result.CustomAlert.TriggerThreshold.ValueFloat64() != 0.05 {
		t.Fatalf("trigger_threshold = %v, want 0.05", result.CustomAlert.TriggerThreshold.ValueFloat64())
	}
	if result.CustomAlert.MinThreshold.ValueFloat64() != 20 {
		t.Fatalf("min_threshold = %v, want 20", result.CustomAlert.MinThreshold.ValueFloat64())
	}
	if result.CustomAlert.Formula == nil || result.CustomAlert.Formula.Right.ValueString() != "requests" {
		t.Fatalf("formula = %#v", result.CustomAlert.Formula)
	}
	if len(result.CustomAlert.Rollups.Elements()) != 2 {
		t.Fatalf("rollups = %#v", result.CustomAlert.Rollups)
	}
	// group_by is null rather than an empty list when the API does not set it,
	// so that it matches a config that omits it.
	if !result.CustomAlert.GroupBy.IsNull() {
		t.Fatalf("group_by = %#v, want null", result.CustomAlert.GroupBy)
	}
}

func TestAlertRuleCustomAlertRoundTripsThroughClientModel(t *testing.T) {
	ctx := context.Background()
	configured := &AlertRuleCustomAlert{
		Event: types.StringValue("incomingRequest"),
		Rollups: types.MapValueMust(alertRuleRollupAttrType, map[string]attr.Value{
			"requests": types.ObjectValueMust(alertRuleRollupAttrType.AttrTypes, map[string]attr.Value{
				"measure":     types.StringValue("count"),
				"aggregation": types.StringValue("sum"),
				"filter":      types.StringNull(),
			}),
		}),
		GroupBy:          types.ListValueMust(types.StringType, []attr.Value{types.StringValue("route")}),
		Filter:           types.StringValue("route ne '/health'"),
		Granularity:      types.StringValue("5m"),
		TriggerType:      types.StringValue("anomaly"),
		TriggerOperator:  types.StringValue("gt"),
		TriggerThreshold: types.Float64Value(3),
		MinThreshold:     types.Float64Value(10),
	}

	converted, diags := alertRuleCustomAlertToClient(ctx, configured)
	if diags.HasError() {
		t.Fatalf("alertRuleCustomAlertToClient() diagnostics = %v", diags)
	}

	result, diags := alertRuleCustomAlertFromClient(ctx, converted)
	if diags.HasError() {
		t.Fatalf("alertRuleCustomAlertFromClient() diagnostics = %v", diags)
	}

	if !result.Event.Equal(configured.Event) ||
		!result.Rollups.Equal(configured.Rollups) ||
		!result.GroupBy.Equal(configured.GroupBy) ||
		!result.Filter.Equal(configured.Filter) ||
		!result.Granularity.Equal(configured.Granularity) ||
		!result.TriggerType.Equal(configured.TriggerType) ||
		!result.TriggerOperator.Equal(configured.TriggerOperator) ||
		!result.TriggerThreshold.Equal(configured.TriggerThreshold) ||
		!result.MinThreshold.Equal(configured.MinThreshold) {
		t.Fatalf("custom alert round trip = %#v, want %#v", result, configured)
	}
	if result.Formula != nil {
		t.Fatalf("formula = %#v, want nil", result.Formula)
	}
}

func TestAlertRuleProjectTargetPrefersProjectID(t *testing.T) {
	custom := AlertRule{
		ProjectID:     types.StringValue("prj_123"),
		ProjectFilter: types.StringNull(),
	}
	if target := custom.projectTarget(); target == nil || *target != "prj_123" {
		t.Fatalf("projectTarget() = %v, want prj_123", target)
	}

	builtIn := AlertRule{
		ProjectID:     types.StringNull(),
		ProjectFilter: types.StringValue("projectId in ('prj_123')"),
	}
	if target := builtIn.projectTarget(); target == nil || *target != "projectId in ('prj_123')" {
		t.Fatalf("projectTarget() = %v, want the OData filter", target)
	}

	teamWide := AlertRule{
		ProjectID:     types.StringNull(),
		ProjectFilter: types.StringNull(),
	}
	if target := teamWide.projectTarget(); target != nil {
		t.Fatalf("projectTarget() = %v, want nil for a team-wide rule", *target)
	}
}
