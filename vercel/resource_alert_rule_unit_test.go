package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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

	triggerMode := rule.Attributes["trigger_mode"].(schema.StringAttribute)
	notificationSettings := rule.Attributes["notification_settings"].(schema.SingleNestedAttribute)
	isDefault := rule.Attributes["is_default"].(schema.BoolAttribute)
	createdAt := rule.Attributes["created_at"].(schema.Int64Attribute)
	if len(triggerMode.PlanModifiers) == 0 || len(notificationSettings.PlanModifiers) == 0 || len(isDefault.PlanModifiers) == 0 || len(createdAt.PlanModifiers) == 0 {
		t.Fatal("stable computed alert rule attributes must retain known state during updates")
	}
}

func TestAlertRuleConfiguredTriggersPlanSelectedMode(t *testing.T) {
	ctx := context.Background()
	ruleSchema := alertRuleSchema(t)
	configuredTriggers := alertRuleTriggerSet(t, "statusGroup:5xx")
	scope := alertRuleScopeValue("all", types.SetNull(types.StringType))
	notificationSettings := types.ObjectValueMust(alertRuleNotificationSettingsAttrType.AttrTypes, map[string]attr.Value{
		"enable_team_owner_notifications": types.BoolValue(true),
		"incident_io_routing_key":         types.StringNull(),
	})

	config := AlertRule{
		ID: types.StringNull(), TeamID: types.StringNull(), Type: types.StringValue(client.AlertRuleTypeBuiltIn), Name: types.StringValue("Errors"),
		RuleScope: scope, TriggerMode: types.StringNull(), Triggers: configuredTriggers, MatchMinimumSeverityLevel: types.StringValue("high"),
		NotificationSettings: types.ObjectNull(alertRuleNotificationSettingsAttrType.AttrTypes), IsDefault: types.BoolNull(), CreatedAt: types.Int64Null(), UpdatedAt: types.Int64Null(),
	}
	state := config
	state.ID = types.StringValue("ar_123")
	state.TeamID = types.StringValue("team_123")
	state.TriggerMode = types.StringValue("all")
	state.Triggers = types.SetNull(alertRuleTriggerAttrType)
	state.NotificationSettings = notificationSettings
	state.IsDefault = types.BoolValue(false)
	state.CreatedAt = types.Int64Value(1)
	state.UpdatedAt = types.Int64Value(1)
	plan := state
	plan.TriggerMode = types.StringUnknown()
	plan.Triggers = configuredTriggers

	configPlan := tfsdk.Plan{Schema: ruleSchema}
	if diags := configPlan.Set(ctx, config); diags.HasError() {
		t.Fatalf("config Plan.Set() diagnostics = %v", diags)
	}
	plannedState := tfsdk.Plan{Schema: ruleSchema}
	if diags := plannedState.Set(ctx, plan); diags.HasError() {
		t.Fatalf("plan Set() diagnostics = %v", diags)
	}
	priorState := tfsdk.State{Schema: ruleSchema}
	if diags := priorState.Set(ctx, state); diags.HasError() {
		t.Fatalf("state Set() diagnostics = %v", diags)
	}

	response := &resource.ModifyPlanResponse{Plan: plannedState}
	(&alertRuleResource{}).ModifyPlan(ctx, resource.ModifyPlanRequest{
		Config: tfsdk.Config{Raw: configPlan.Raw, Schema: ruleSchema},
		Plan:   plannedState,
		State:  priorState,
	}, response)
	if response.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan() diagnostics = %v", response.Diagnostics)
	}

	var modified AlertRule
	if diags := response.Plan.Get(ctx, &modified); diags.HasError() {
		t.Fatalf("modified Plan.Get() diagnostics = %v", diags)
	}
	if got := modified.TriggerMode.ValueString(); got != "selected" {
		t.Fatalf("trigger_mode = %q, want selected", got)
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

func TestAlertRuleUnknownNestedConfigurationIsDeferred(t *testing.T) {
	ctx := context.Background()
	ruleSchema := alertRuleSchema(t)

	tests := map[string]func(AlertRule) AlertRule{
		"entire scope object": func(config AlertRule) AlertRule {
			config.RuleScope = types.ObjectUnknown(alertRuleScopeAttrType.AttrTypes)
			return config
		},
		"scope project IDs": func(config AlertRule) AlertRule {
			config.RuleScope = alertRuleScopeValue("all", types.SetUnknown(types.StringType))
			return config
		},
		"entire trigger object": func(config AlertRule) AlertRule {
			config.Triggers = types.SetValueMust(alertRuleTriggerAttrType, []attr.Value{
				types.ObjectUnknown(alertRuleTriggerAttrType.AttrTypes),
			})
			return config
		},
		"trigger filter": func(config AlertRule) AlertRule {
			config.Triggers = types.SetValueMust(alertRuleTriggerAttrType, []attr.Value{
				types.ObjectValueMust(alertRuleTriggerAttrType.AttrTypes, map[string]attr.Value{
					"type":   types.StringValue("botId"),
					"filter": types.StringUnknown(),
				}),
			})
			return config
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			config := mutate(alertRuleConfiguration(t))
			configPlan := tfsdk.Plan{Schema: ruleSchema}
			if diags := configPlan.Set(ctx, config); diags.HasError() {
				t.Fatalf("config Plan.Set() diagnostics = %v", diags)
			}

			response := &resource.ValidateConfigResponse{}
			(&alertRuleResource{}).ValidateConfig(ctx, resource.ValidateConfigRequest{
				Config: tfsdk.Config{Raw: configPlan.Raw, Schema: ruleSchema},
			}, response)
			if response.Diagnostics.HasError() {
				t.Fatalf("ValidateConfig() diagnostics = %v", response.Diagnostics)
			}
		})
	}
}

func TestAlertRuleKnownInvalidNestedConfigurationIsRejected(t *testing.T) {
	ctx := context.Background()
	ruleSchema := alertRuleSchema(t)

	tests := map[string]func(AlertRule) AlertRule{
		"all scope with project IDs": func(config AlertRule) AlertRule {
			config.RuleScope = alertRuleScopeValue("all", types.SetValueMust(types.StringType, []attr.Value{types.StringValue("prj_123")}))
			return config
		},
		"include scope without project IDs": func(config AlertRule) AlertRule {
			config.RuleScope = alertRuleScopeValue("include", types.SetNull(types.StringType))
			return config
		},
		"unsupported trigger filter": func(config AlertRule) AlertRule {
			config.Triggers = types.SetValueMust(alertRuleTriggerAttrType, []attr.Value{
				types.ObjectValueMust(alertRuleTriggerAttrType.AttrTypes, map[string]attr.Value{
					"type":   types.StringValue("botId"),
					"filter": types.StringValue("botId:bot_123"),
				}),
			})
			return config
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			config := mutate(alertRuleConfiguration(t))
			configPlan := tfsdk.Plan{Schema: ruleSchema}
			if diags := configPlan.Set(ctx, config); diags.HasError() {
				t.Fatalf("config Plan.Set() diagnostics = %v", diags)
			}

			response := &resource.ValidateConfigResponse{}
			(&alertRuleResource{}).ValidateConfig(ctx, resource.ValidateConfigRequest{
				Config: tfsdk.Config{Raw: configPlan.Raw, Schema: ruleSchema},
			}, response)
			if !response.Diagnostics.HasError() {
				t.Fatal("ValidateConfig() returned no diagnostics")
			}
		})
	}
}

func TestAlertRuleModifyPlanAcceptsUnknownScopeObject(t *testing.T) {
	ctx := context.Background()
	ruleSchema := alertRuleSchema(t)
	state := alertRuleConfiguration(t)
	state.ID = types.StringValue("ar_123")
	state.TeamID = types.StringValue("team_123")
	state.TriggerMode = types.StringValue("selected")
	state.NotificationSettings = types.ObjectValueMust(alertRuleNotificationSettingsAttrType.AttrTypes, map[string]attr.Value{
		"enable_team_owner_notifications": types.BoolValue(true),
		"incident_io_routing_key":         types.StringNull(),
	})
	state.IsDefault = types.BoolValue(false)
	state.CreatedAt = types.Int64Value(1)
	state.UpdatedAt = types.Int64Value(1)

	config := state
	config.RuleScope = types.ObjectUnknown(alertRuleScopeAttrType.AttrTypes)
	plan := config
	plan.TriggerMode = types.StringUnknown()

	configPlan := tfsdk.Plan{Schema: ruleSchema}
	if diags := configPlan.Set(ctx, config); diags.HasError() {
		t.Fatalf("config Plan.Set() diagnostics = %v", diags)
	}
	plannedState := tfsdk.Plan{Schema: ruleSchema}
	if diags := plannedState.Set(ctx, plan); diags.HasError() {
		t.Fatalf("plan Set() diagnostics = %v", diags)
	}
	priorState := tfsdk.State{Schema: ruleSchema}
	if diags := priorState.Set(ctx, state); diags.HasError() {
		t.Fatalf("state Set() diagnostics = %v", diags)
	}

	response := &resource.ModifyPlanResponse{Plan: plannedState}
	(&alertRuleResource{}).ModifyPlan(ctx, resource.ModifyPlanRequest{
		Config: tfsdk.Config{Raw: configPlan.Raw, Schema: ruleSchema},
		Plan:   plannedState,
		State:  priorState,
	}, response)
	if response.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan() diagnostics = %v", response.Diagnostics)
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
	scope := alertRuleScopeModel(t, rule.RuleScope)
	if rule.Type.ValueString() != client.AlertRuleTypeBuiltIn || scope.Type.ValueString() != "include" || rule.TriggerMode.ValueString() != "selected" {
		t.Fatalf("rule = %#v", rule)
	}
	if len(rule.Triggers.Elements()) != 1 || len(scope.ProjectIDs.Elements()) != 1 {
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

func TestAlertRuleUpdatePreservesUnconfiguredOwnerNotifications(t *testing.T) {
	ctx := context.Background()
	notificationSettings := alertRuleSchema(t).Attributes["notification_settings"].(schema.SingleNestedAttribute)
	enableOwners := notificationSettings.Attributes["enable_team_owner_notifications"].(schema.BoolAttribute)
	if len(enableOwners.PlanModifiers) != 1 {
		t.Fatalf("enable_team_owner_notifications plan modifiers = %d, want 1", len(enableOwners.PlanModifiers))
	}

	modifierResponse := &planmodifier.BoolResponse{PlanValue: types.BoolUnknown()}
	enableOwners.PlanModifiers[0].PlanModifyBool(ctx, planmodifier.BoolRequest{
		ConfigValue: types.BoolNull(),
		PlanValue:   types.BoolUnknown(),
		StateValue:  types.BoolValue(false),
	}, modifierResponse)
	if modifierResponse.Diagnostics.HasError() {
		t.Fatalf("PlanModifyBool() diagnostics = %v", modifierResponse.Diagnostics)
	}
	if modifierResponse.PlanValue.IsUnknown() || modifierResponse.PlanValue.ValueBool() {
		t.Fatalf("planned enable_team_owner_notifications = %v, want false", modifierResponse.PlanValue)
	}

	filter := "statusGroup:5xx"
	severity := "high"
	routingKey := "checkout"
	state, diags := alertRuleFromAPI(ctx, client.AlertRule{
		ID: "ar_123", Type: client.AlertRuleTypeBuiltIn, Name: "Errors",
		RuleScope:                 client.AlertRuleScope{Type: "all"},
		Triggers:                  &client.AlertRuleTriggers{Mode: "selected", Items: []client.AlertRuleTrigger{{Type: "error_anomaly", Filter: &filter}}},
		MatchMinimumSeverityLevel: &severity,
		NotificationSettings: client.AlertRuleNotificationSettings{
			EnableTeamOwnerNotifications: false,
			IncidentIORoutingKey:         &routingKey,
		},
	}, types.StringValue("team_123"))
	if diags.HasError() {
		t.Fatalf("alertRuleFromAPI() diagnostics = %v", diags)
	}
	plan := state
	plan.Name = types.StringValue("Renamed")
	plan.NotificationSettings = types.ObjectValueMust(alertRuleNotificationSettingsAttrType.AttrTypes, map[string]attr.Value{
		"enable_team_owner_notifications": modifierResponse.PlanValue,
		"incident_io_routing_key":         types.StringValue(routingKey),
	})

	request, diags := plan.toUpdateRequest(ctx, state)
	if diags.HasError() {
		t.Fatalf("toUpdateRequest() diagnostics = %v", diags)
	}
	if request.Name == nil || *request.Name != "Renamed" {
		t.Fatalf("request.Name = %#v, want Renamed", request.Name)
	}
	if request.NotificationSettings != nil {
		t.Fatalf("request.NotificationSettings = %#v, want omitted", request.NotificationSettings)
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

func alertRuleConfiguration(t *testing.T) AlertRule {
	t.Helper()
	return AlertRule{
		ID:                        types.StringNull(),
		TeamID:                    types.StringNull(),
		Type:                      types.StringValue(client.AlertRuleTypeBuiltIn),
		Name:                      types.StringValue("Errors"),
		RuleScope:                 alertRuleScopeValue("all", types.SetNull(types.StringType)),
		TriggerMode:               types.StringNull(),
		Triggers:                  alertRuleTriggerSet(t, "statusGroup:5xx"),
		MatchMinimumSeverityLevel: types.StringValue("high"),
		NotificationSettings:      types.ObjectNull(alertRuleNotificationSettingsAttrType.AttrTypes),
		IsDefault:                 types.BoolNull(),
		CreatedAt:                 types.Int64Null(),
		UpdatedAt:                 types.Int64Null(),
	}
}

func alertRuleScopeValue(scopeType string, projectIDs types.Set) types.Object {
	return types.ObjectValueMust(alertRuleScopeAttrType.AttrTypes, map[string]attr.Value{
		"type":        types.StringValue(scopeType),
		"project_ids": projectIDs,
	})
}

func alertRuleScopeModel(t *testing.T, value types.Object) AlertRuleScope {
	t.Helper()
	var scope AlertRuleScope
	diags := value.As(context.Background(), &scope, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		t.Fatalf("Object.As() diagnostics = %v", diags)
	}
	return scope
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
