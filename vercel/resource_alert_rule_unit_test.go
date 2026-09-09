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
	for _, name := range []string{"type", "name", "rule_scope", "match_minimum_severity_level"} {
		if !rule.Attributes[name].IsRequired() {
			t.Fatalf("%s must be required", name)
		}
	}
	if !rule.Attributes["triggers"].IsOptional() || !rule.Attributes["triggers"].IsComputed() || !rule.Attributes["trigger_mode"].IsComputed() {
		t.Fatal("triggers must be optional and computed, and trigger mode must be computed")
	}
	if !rule.Attributes["notification_settings"].IsOptional() || !rule.Attributes["notification_settings"].IsComputed() || !rule.Attributes["is_default"].IsComputed() {
		t.Fatal("notification settings must be optional and computed, and default metadata must be computed")
	}
}

func TestAlertRuleUnknownNotificationSettingsAreOmitted(t *testing.T) {
	settings, diags := alertRuleNotificationSettingsToClient(
		context.Background(),
		types.ObjectUnknown(alertRuleNotificationSettingsAttrType.AttrTypes),
	)
	if diags.HasError() {
		t.Fatalf("alertRuleNotificationSettingsToClient() diagnostics = %v", diags)
	}
	if settings != nil {
		t.Fatalf("settings = %#v, want nil", settings)
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
	if rule.Type.ValueString() != client.AlertRuleTypeBuiltIn || rule.RuleScope.Type.ValueString() != "include" || rule.TriggerMode.ValueString() != "selected" {
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

func TestAlertRuleAllTriggerModeDoesNotFabricateTriggerState(t *testing.T) {
	triggers, diags := alertRuleTriggersFromClient(context.Background(), &client.AlertRuleTriggers{Mode: "all"})
	if diags.HasError() {
		t.Fatalf("alertRuleTriggersFromClient() diagnostics = %v", diags)
	}
	if !triggers.IsNull() {
		t.Fatalf("triggers = %#v, want null for response-only all mode", triggers)
	}
}

func TestAlertRuleEmptySelectedTriggerStateIsPreserved(t *testing.T) {
	triggers, diags := alertRuleTriggersFromClient(context.Background(), &client.AlertRuleTriggers{Mode: "selected", Items: []client.AlertRuleTrigger{}})
	if diags.HasError() {
		t.Fatalf("alertRuleTriggersFromClient() diagnostics = %v", diags)
	}
	if triggers.IsNull() || len(triggers.Elements()) != 0 {
		t.Fatalf("triggers = %#v, want a known empty set", triggers)
	}
}

func TestAlertRuleUpdateOmitsUnchangedLegacyTriggers(t *testing.T) {
	ctx := context.Background()
	filter := "statusGroup:5xx OR route:/api"
	severity := "high"
	state, diags := alertRuleFromAPI(ctx, client.AlertRule{
		ID: "ar_123", Type: client.AlertRuleTypeBuiltIn, Name: "Errors",
		RuleScope:                 client.AlertRuleScope{Type: "all"},
		Triggers:                  &client.AlertRuleTriggers{Mode: "selected", Items: []client.AlertRuleTrigger{{Type: "error_anomaly", Filter: &filter}}},
		MatchMinimumSeverityLevel: &severity,
		NotificationSettings:      client.AlertRuleNotificationSettings{EnableTeamOwnerNotifications: true},
	}, types.StringValue("team_123"))
	if diags.HasError() {
		t.Fatalf("alertRuleFromAPI() diagnostics = %v", diags)
	}
	plan := state
	plan.Name = types.StringValue("Renamed")

	request, diags := plan.toUpdateRequest(ctx, state)
	if diags.HasError() {
		t.Fatalf("toUpdateRequest() diagnostics = %v", diags)
	}
	if request.Name == nil || *request.Name != "Renamed" {
		t.Fatalf("request.Name = %#v, want Renamed", request.Name)
	}
	if request.Type != nil || request.RuleScope != nil || request.Triggers != nil || request.MatchMinimumSeverityLevel != nil || request.NotificationSettings != nil {
		t.Fatalf("unchanged fields were included in request: %#v", request)
	}
}

func TestAlertRuleTriggerFiltersPreserveConfiguredRepresentation(t *testing.T) {
	configured := alertRuleTriggerSet(t, "NOT statusGroup:4xx")
	canonical := alertRuleTriggerSet(t, "statusGroup:5xx")
	canonicalFilter := "statusGroup:5xx"

	created, diags := alertRuleTriggersPreservingFilters(
		context.Background(),
		canonical,
		configured,
		nil,
		alertRuleCanonicalFilters{"error_anomaly": &canonicalFilter},
		alertRuleFilterReconcileApply,
	)
	if diags.HasError() {
		t.Fatalf("alertRuleTriggersPreservingFilters() diagnostics = %v", diags)
	}
	if got := alertRuleTriggerFilter(t, created); got != "NOT statusGroup:4xx" {
		t.Fatalf("created filter = %q, want configured representation", got)
	}

	refreshed, diags := alertRuleTriggersPreservingFilters(
		context.Background(),
		canonical,
		created,
		alertRuleCanonicalFilters{"error_anomaly": &canonicalFilter},
		alertRuleCanonicalFilters{"error_anomaly": &canonicalFilter},
		alertRuleFilterReconcileRefresh,
	)
	if diags.HasError() {
		t.Fatalf("alertRuleTriggersPreservingFilters() diagnostics = %v", diags)
	}
	if got := alertRuleTriggerFilter(t, refreshed); got != "NOT statusGroup:4xx" {
		t.Fatalf("refreshed filter = %q, want configured representation", got)
	}
}

func TestAlertRuleTriggerFiltersExposeRemoteDrift(t *testing.T) {
	configured := alertRuleTriggerSet(t, "NOT statusGroup:4xx")
	remote := alertRuleTriggerSet(t, "statusGroup:4xx")
	previousCanonicalFilter := "statusGroup:5xx"
	currentCanonicalFilter := "statusGroup:4xx"

	refreshed, diags := alertRuleTriggersPreservingFilters(
		context.Background(),
		remote,
		configured,
		alertRuleCanonicalFilters{"error_anomaly": &previousCanonicalFilter},
		alertRuleCanonicalFilters{"error_anomaly": &currentCanonicalFilter},
		alertRuleFilterReconcileRefresh,
	)
	if diags.HasError() {
		t.Fatalf("alertRuleTriggersPreservingFilters() diagnostics = %v", diags)
	}
	if got := alertRuleTriggerFilter(t, refreshed); got != "statusGroup:4xx" {
		t.Fatalf("refreshed filter = %q, want remote value", got)
	}
}

func TestAlertRuleImportRequiresTeam(t *testing.T) {
	response := &resource.ImportStateResponse{}
	(&alertRuleResource{client: client.New("TOKEN")}).ImportState(
		context.Background(),
		resource.ImportStateRequest{ID: "ar_123"},
		response,
	)
	if !response.Diagnostics.HasError() {
		t.Fatal("ImportState() returned no diagnostics without team context")
	}
}

func alertRuleTriggerSet(t *testing.T, filter string) types.Set {
	t.Helper()
	value, diags := types.SetValueFrom(context.Background(), alertRuleTriggerAttrType, []AlertRuleTrigger{{
		Type:   types.StringValue("error_anomaly"),
		Filter: types.StringValue(filter),
	}})
	if diags.HasError() {
		t.Fatalf("types.SetValueFrom() diagnostics = %v", diags)
	}
	return value
}

func alertRuleTriggerFilter(t *testing.T, value types.Set) string {
	t.Helper()
	var triggers []AlertRuleTrigger
	diags := value.ElementsAs(context.Background(), &triggers, false)
	if diags.HasError() {
		t.Fatalf("ElementsAs() diagnostics = %v", diags)
	}
	if len(triggers) != 1 {
		t.Fatalf("trigger count = %d, want 1", len(triggers))
	}
	return triggers[0].Filter.ValueString()
}
