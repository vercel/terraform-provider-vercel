package vercel

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func monitorSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	var response resource.SchemaResponse
	newMonitorResource().Schema(context.Background(), resource.SchemaRequest{}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics = %v", response.Diagnostics)
	}
	return response
}

func TestMonitorSchemaMirrorsAlertRule(t *testing.T) {
	monitor := monitorSchema(t).Schema
	rule := alertRuleSchema(t)

	if len(monitor.Attributes) != len(rule.Attributes) {
		t.Fatalf("monitor has %d attributes, alert rule has %d", len(monitor.Attributes), len(rule.Attributes))
	}
	for name, attribute := range rule.Attributes {
		got, ok := monitor.Attributes[name]
		if !ok {
			t.Fatalf("monitor schema is missing %q", name)
		}
		if got.IsRequired() != attribute.IsRequired() || got.IsOptional() != attribute.IsOptional() || got.IsComputed() != attribute.IsComputed() {
			t.Fatalf("%s flags differ from alert rule", name)
		}
	}
	if monitor.MarkdownDescription == rule.MarkdownDescription {
		t.Fatal("monitor schema description must not be the alert rule description")
	}
}

func TestMonitorSchemaDescriptionsUseMonitorWording(t *testing.T) {
	monitor := monitorSchema(t).Schema
	rule := alertRuleSchema(t)

	for _, name := range []string{"id", "team_id", "type", "name", "rule_scope", "tags", "notification_settings", "is_default"} {
		got := monitor.Attributes[name].GetMarkdownDescription()
		if got == rule.Attributes[name].GetMarkdownDescription() {
			t.Errorf("%s description was not overridden for monitors", name)
		}
		if strings.Contains(got, "alert rule") || strings.Contains(got, "the rule") {
			t.Errorf("%s description %q still uses alert rule wording", name, got)
		}
	}
}

func TestMonitorMetadata(t *testing.T) {
	var response resource.MetadataResponse
	newMonitorResource().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "vercel"}, &response)
	if response.TypeName != "vercel_monitor" {
		t.Fatalf("TypeName = %q, want vercel_monitor", response.TypeName)
	}
}

func TestMonitorCreateRequestUsesMonitorsAPIAndInjectsProviderTag(t *testing.T) {
	ctx := context.Background()
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/alerts/v3/monitors" {
			t.Fatalf("request = %s %s, want POST /alerts/v3/monitors", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"monitor":{"id":"ar_123","type":"built-in","name":"Errors","ruleScope":{"type":"all"},"triggers":{"mode":"selected","items":[{"type":"error_anomaly","filter":"statusGroup:5xx"}]},"matchMinimumSeverityLevel":"high","tags":["checkout","terraform-provider-vercel"],"notificationSettings":{"enableTeamOwnerNotifications":true},"isDefault":false}}`))
	}))
	t.Cleanup(server.Close)

	config := alertRuleConfiguration(t)
	config.Tags = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("checkout")})
	payload, diags := config.toCreateRequest(ctx)
	if diags.HasError() {
		t.Fatalf("toCreateRequest() diagnostics = %v", diags)
	}

	out, err := client.New("TOKEN").WithBaseURL(server.URL).CreateMonitor(ctx, client.CreateAlertRuleRequest{TeamID: "team_123", AlertRuleCreate: payload})
	if err != nil {
		t.Fatalf("CreateMonitor() error = %v", err)
	}

	var tags []string
	for _, tag := range body["tags"].([]any) {
		tags = append(tags, tag.(string))
	}
	if !slices.Contains(tags, terraformProviderVercelAlertRuleTag) || !slices.Contains(tags, "checkout") {
		t.Fatalf("request tags = %v, want checkout and %s", tags, terraformProviderVercelAlertRuleTag)
	}

	state, diags := alertRuleFromAPI(ctx, out, types.StringValue("team_123"), config.Tags)
	if diags.HasError() {
		t.Fatalf("alertRuleFromAPI() diagnostics = %v", diags)
	}
	var stateTags []string
	state.Tags.ElementsAs(ctx, &stateTags, false)
	if state.ID.ValueString() != "ar_123" || slices.Contains(stateTags, terraformProviderVercelAlertRuleTag) {
		t.Fatalf("state = %#v, want id ar_123 with provider tag hidden", state)
	}
}

func TestMonitorModifyPlanRequiresTriggersOnCreate(t *testing.T) {
	ctx := context.Background()
	monitorSchema := monitorSchema(t).Schema
	config := alertRuleConfiguration(t)
	config.Triggers = types.SetNull(alertRuleTriggerAttrType)
	plan := tfsdk.Plan{Schema: monitorSchema}
	if diags := plan.Set(ctx, config); diags.HasError() {
		t.Fatalf("Plan.Set() diagnostics = %v", diags)
	}

	response := &resource.ModifyPlanResponse{Plan: plan}
	(&monitorResource{}).ModifyPlan(ctx, resource.ModifyPlanRequest{
		Config: tfsdk.Config{Raw: plan.Raw, Schema: monitorSchema},
		Plan:   plan,
		State:  tfsdk.State{Schema: monitorSchema, Raw: tftypes.NewValue(monitorSchema.Type().TerraformType(ctx), nil)},
	}, response)
	if !response.Diagnostics.HasError() {
		t.Fatal("ModifyPlan() returned no diagnostics for a monitor without triggers")
	}
	if got := response.Diagnostics.Errors()[0].Summary(); got != "Missing built-in monitor triggers" {
		t.Fatalf("ModifyPlan() summary = %q, want monitor wording", got)
	}
}

func TestMonitorModifyPlanRejectsCriticalMatchLevel(t *testing.T) {
	ctx := context.Background()
	monitorSchema := monitorSchema(t).Schema
	config := alertRuleConfiguration(t)
	config.MatchMinimumSeverityLevel = types.StringValue("critical")
	plan := tfsdk.Plan{Schema: monitorSchema}
	if diags := plan.Set(ctx, config); diags.HasError() {
		t.Fatalf("Plan.Set() diagnostics = %v", diags)
	}

	response := &resource.ModifyPlanResponse{Plan: plan}
	(&monitorResource{}).ModifyPlan(ctx, resource.ModifyPlanRequest{
		Config: tfsdk.Config{Raw: plan.Raw, Schema: monitorSchema},
		Plan:   plan,
		State:  tfsdk.State{Schema: monitorSchema, Raw: tftypes.NewValue(monitorSchema.Type().TerraformType(ctx), nil)},
	}, response)
	if !response.Diagnostics.HasError() {
		t.Fatal("ModifyPlan() returned no diagnostics for a critical match level")
	}
	if got := response.Diagnostics.Errors()[0].Summary(); got != "Critical-only built-in monitors are not supported" {
		t.Fatalf("ModifyPlan() summary = %q, want monitor wording", got)
	}
}

func TestMonitorValidateConfigUsesMonitorWording(t *testing.T) {
	ctx := context.Background()
	monitorSchema := monitorSchema(t).Schema
	config := alertRuleConfiguration(t)
	config.RuleScope = alertRuleScopeValue("include", types.SetNull(types.StringType))
	raw := tfsdk.Config{Schema: monitorSchema}
	state := tfsdk.State{Schema: monitorSchema}
	if diags := state.Set(ctx, config); diags.HasError() {
		t.Fatalf("State.Set() diagnostics = %v", diags)
	}
	raw.Raw = state.Raw

	response := &resource.ValidateConfigResponse{}
	(&monitorResource{}).ValidateConfig(ctx, resource.ValidateConfigRequest{Config: raw}, response)
	if !response.Diagnostics.HasError() {
		t.Fatal("ValidateConfig() returned no diagnostics for an include scope without project_ids")
	}
	if got := response.Diagnostics.Errors()[0].Summary(); got != "Invalid built-in monitor scope" {
		t.Fatalf("ValidateConfig() summary = %q, want monitor wording", got)
	}
}

func TestMonitorImportRequiresTeam(t *testing.T) {
	response := &resource.ImportStateResponse{}
	(&monitorResource{client: client.New("TOKEN")}).ImportState(
		context.Background(),
		resource.ImportStateRequest{ID: "ar_123"},
		response,
	)
	if !response.Diagnostics.HasError() {
		t.Fatal("ImportState() returned no diagnostics without team context")
	}
}
