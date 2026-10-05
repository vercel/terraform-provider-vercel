package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func customAlertRuleSchema(t *testing.T) schema.Schema {
	t.Helper()
	var response resource.SchemaResponse
	newCustomAlertRuleResource().Schema(context.Background(), resource.SchemaRequest{}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	return response.Schema
}

func customAlertEvaluationValue(window, query string) types.Object {
	value, diags := customAlertQueryFromJSON(context.Background(), json.RawMessage(query))
	if diags.HasError() {
		panic(diags)
	}
	return types.ObjectValueMust(customAlertEvaluationAttrTypes, map[string]attr.Value{
		"window": types.StringValue(window), "query": value,
	})
}

func customAlertThresholdValue(threshold float64) types.Object {
	return types.ObjectValueMust(customAlertTriggerAttrTypes, map[string]attr.Value{
		"type": types.StringValue("threshold"), "output": types.StringValue("requests"),
		"operator": types.StringValue("gt"), "threshold": types.Float64Value(threshold),
		"standard_deviations": types.Float64Null(), "minimum": types.ObjectNull(customAlertMinimumAttrTypes),
	})
}

func customAlertTestModel() CustomAlertRule {
	return CustomAlertRule{
		ID: types.StringValue("ar_123"), TeamID: types.StringValue("team_123"), Name: types.StringValue("Requests"),
		ProjectID: types.StringValue("prj_123"), Severity: types.StringValue("high"),
		Evaluation: customAlertEvaluationValue("5m", `{"metrics":{"requests":{"metric":"vercel.request.count","aggregation":"count"}},"outputs":["requests"]}`),
		Trigger:    customAlertThresholdValue(0), InvestigationPrompt: types.StringNull(), Tags: types.SetNull(types.StringType),
		NotificationSettings: types.ObjectNull(alertRuleNotificationSettingsAttrType.AttrTypes), CreatedAt: types.Int64Null(), UpdatedAt: types.Int64Null(),
	}
}

func TestCustomAlertRuleRequests(t *testing.T) {
	ctx := context.Background()
	state := customAlertTestModel()
	create, diags := state.toCreateRequest(ctx)
	if diags.HasError() {
		t.Fatal(diags)
	}
	encoded, err := json.Marshal(create)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(encoded, &body); err != nil {
		t.Fatal(err)
	}
	if body["type"] != "custom" || create.Trigger.Threshold == nil || *create.Trigger.Threshold != 0 {
		t.Fatalf("create = %s", encoded)
	}
	for _, field := range []string{"teamId", "id", "triggers", "matchMinimumSeverityLevel", "investigationPrompt", "agentTriageEnabled", "tags"} {
		if _, exists := body[field]; exists {
			t.Fatalf("unexpected %s in %s", field, encoded)
		}
	}
	state.InvestigationPrompt = types.StringValue("Investigate checkout failures")
	plan := state
	plan.Name = types.StringValue("Checkout requests")
	plan.InvestigationPrompt = types.StringNull()
	update, diags := plan.toUpdateRequest(ctx, state)
	if diags.HasError() {
		t.Fatal(diags)
	}
	encoded, err = json.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"investigationPrompt":null,"name":"Checkout requests"}` {
		t.Fatalf("metadata PATCH = %s", encoded)
	}
	plan = state
	plan.Evaluation = customAlertEvaluationValue("1h", `{"metrics":{"requests":{"metric":"vercel.request.count","aggregation":"count"}},"outputs":["requests"]}`)
	update, diags = plan.toUpdateRequest(ctx, state)
	if diags.HasError() || update.Evaluation == nil || update.Evaluation.Window != "1h" || update.Trigger != nil || update.RuleScope != nil {
		t.Fatalf("evaluation PATCH = %#v, diagnostics = %v", update, diags)
	}
}

func TestCustomAlertRuleTagsAndNotificationSettingsRequests(t *testing.T) {
	ctx := context.Background()
	state := customAlertTestModel()
	state.Tags = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("checkout")})
	routingKey := "checkout"
	critical := "critical"
	state.NotificationSettings, _ = alertRuleNotificationSettingsFromClient(ctx, client.AlertRuleNotificationSettings{
		EnableTeamOwnerNotifications: true, IncidentIORoutingKey: &routingKey, VercelNotificationsMinimumSeverityLevel: &critical,
	})
	create, diags := state.toCreateRequest(ctx)
	if diags.HasError() {
		t.Fatal(diags)
	}
	encoded, err := json.Marshal(create)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"tags":["checkout"]`) || !strings.Contains(string(encoded), `"notificationSettings":{"enableTeamOwnerNotifications":true,"incidentIoRoutingKey":"checkout","vercelNotificationsMinimumSeverityLevel":"critical"}`) {
		t.Fatalf("create = %s", encoded)
	}

	plan := state
	plan.Tags = types.SetNull(types.StringType)
	plan.NotificationSettings, _ = alertRuleNotificationSettingsFromClient(ctx, client.AlertRuleNotificationSettings{EnableTeamOwnerNotifications: true})
	update, diags := plan.toUpdateRequest(ctx, state)
	if diags.HasError() {
		t.Fatal(diags)
	}
	encoded, err = json.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"tags":null,"notificationSettings":{"enableTeamOwnerNotifications":true,"incidentIoRoutingKey":null,"vercelNotificationsMinimumSeverityLevel":null}}` {
		t.Fatalf("clearing PATCH = %s", encoded)
	}

	rule := client.AlertRule{
		ID: "ar_123", Type: client.AlertRuleTypeCustom, QuerySupported: true, Name: "Requests", Tags: []string{"checkout"},
		RuleScope:  client.AlertRuleScope{Type: "project", ProjectID: "prj_123"},
		Evaluation: &client.CustomAlertEvaluation{Window: "5m", Query: json.RawMessage(`{}`)}, Trigger: &client.CustomAlertTrigger{Type: "threshold", Output: "requests"},
		NotificationSettings: client.AlertRuleNotificationSettings{VercelNotificationsMinimumSeverityLevel: &critical},
	}
	result, diags := customAlertRuleFromAPI(ctx, rule, types.StringValue("team_123"))
	if diags.HasError() {
		t.Fatal(diags)
	}
	if !result.Tags.Equal(state.Tags) || result.NotificationSettings.Attributes()["vercel_notifications_minimum_severity_level"] != types.StringValue("critical") {
		t.Fatalf("result = %#v", result)
	}
	rule.Tags = nil
	if result, _ = customAlertRuleFromAPI(ctx, rule, types.StringValue("team_123")); !result.Tags.IsNull() {
		t.Fatalf("empty tags = %v, want null", result.Tags)
	}
}

func TestCustomAlertRuleTriggerValidation(t *testing.T) {
	ctx := context.Background()
	ruleSchema := customAlertRuleSchema(t)
	tests := []struct {
		name      string
		values    map[string]attr.Value
		wantError bool
	}{
		{name: "zero threshold"},
		{name: "missing operator", values: map[string]attr.Value{"operator": types.StringNull()}, wantError: true},
		{name: "threshold with deviations", values: map[string]attr.Value{"standard_deviations": types.Float64Value(3)}, wantError: true},
		{name: "anomaly with threshold fields", values: map[string]attr.Value{"type": types.StringValue("anomaly"), "standard_deviations": types.Float64Value(3)}, wantError: true},
		{name: "anomaly", values: map[string]attr.Value{"type": types.StringValue("anomaly"), "operator": types.StringNull(), "threshold": types.Float64Null(), "standard_deviations": types.Float64Value(3)}},
		{name: "missing deviations", values: map[string]attr.Value{"type": types.StringValue("anomaly"), "operator": types.StringNull(), "threshold": types.Float64Null()}, wantError: true},
		{name: "unknown operator", values: map[string]attr.Value{"operator": types.StringUnknown()}},
		{name: "unknown type", values: map[string]attr.Value{"type": types.StringUnknown()}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := customAlertTestModel()
			values := model.Trigger.Attributes()
			for key, value := range test.values {
				values[key] = value
			}
			model.Trigger = types.ObjectValueMust(customAlertTriggerAttrTypes, values)
			plan := tfsdk.Plan{Schema: ruleSchema}
			if diags := plan.Set(ctx, model); diags.HasError() {
				t.Fatal(diags)
			}
			var resp resource.ValidateConfigResponse
			(&customAlertRuleResource{}).ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: ruleSchema, Raw: plan.Raw}}, &resp)
			if resp.Diagnostics.HasError() != test.wantError {
				t.Fatalf("diagnostics = %v, want error %v", resp.Diagnostics, test.wantError)
			}
		})
	}
}

func TestCustomAlertRuleQueryReconciliation(t *testing.T) {
	ctx := context.Background()
	authored := `{"metrics":{"requests":{"metric":"vercel.request.count","aggregation":"count","filter":"httpStatus >= 500"}},"outputs":["requests"]}`
	canonical := `{"metrics":{"requests":{"metric":"vercel.request.count","aggregation":"count","filter":"httpStatus:>=500"}},"outputs":["requests"]}`
	drifted := `{"metrics":{"requests":{"metric":"vercel.request.count","aggregation":"count","filter":"httpStatus:>=400"}},"outputs":["requests"]}`
	for _, test := range []struct {
		name              string
		previous, current string
		apply             bool
		expected          string
	}{
		{"apply", "", canonical, true, authored},
		{"unchanged refresh", canonical, canonical, false, authored},
		{"remote drift", canonical, drifted, false, drifted},
		{"import without baseline", "", canonical, false, canonical},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, diags := customAlertEvaluationPreservingQuery(ctx, customAlertEvaluationValue("1h", test.current), customAlertEvaluationValue("5m", authored), []byte(test.previous), []byte(test.current), test.apply)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if !result.Attributes()["query"].Equal(customAlertEvaluationValue("1h", test.expected).Attributes()["query"]) || result.Attributes()["window"] != types.StringValue("1h") {
				t.Fatalf("result = %v", result)
			}
		})
	}
}

func TestCustomAlertRuleRejectsUnsupportedResponses(t *testing.T) {
	for _, rule := range []client.AlertRule{
		{Type: client.AlertRuleTypeBuiltIn},
		{Type: client.AlertRuleTypeCustom, QuerySupported: false},
		{Type: client.AlertRuleTypeCustom, QuerySupported: true},
	} {
		if _, diags := customAlertRuleFromAPI(context.Background(), rule, types.StringValue("team_123")); !diags.HasError() {
			t.Fatalf("accepted unsupported rule %#v", rule)
		}
	}
}

type customAlertMockProvider struct {
	provider.Provider
	client *client.Client
}

func (p *customAlertMockProvider) Configure(_ context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	resp.ResourceData = p.client
	resp.DataSourceData = p.client
}

// Terraform drives the complete lifecycle against a local API, including the
// private query baseline that is unavailable in direct resource method tests.
func TestCustomAlertRuleLifecycle(t *testing.T) {
	t.Run("normal updates", func(t *testing.T) { testCustomAlertRuleLifecycle(t, false) })
	t.Run("concurrent query change", func(t *testing.T) { testCustomAlertRuleLifecycle(t, true) })
}

func testCustomAlertRuleLifecycle(t *testing.T, concurrentQueryChange bool) {
	t.Helper()
	var mu sync.Mutex
	var rule client.AlertRule
	var patches []map[string]json.RawMessage
	var deleted bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if req.URL.Query().Get("teamId") != "team_123" {
			t.Errorf("teamId = %q", req.URL.Query().Get("teamId"))
			http.Error(w, "wrong team", 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case http.MethodPost:
			if req.URL.Path != "/alerts/v3/alert-rules" {
				t.Errorf("create path = %s", req.URL.Path)
			}
			if err := json.NewDecoder(req.Body).Decode(&rule); err != nil {
				t.Error(err)
				http.Error(w, "invalid", 400)
				return
			}
			rule.ID, rule.QuerySupported = "ar_123", true
			rule.NotificationSettings.EnableTeamOwnerNotifications = true
			// Simulate the API's KQL canonicalization.
			var query map[string]any
			_ = json.Unmarshal(rule.Evaluation.Query, &query)
			query["metrics"].(map[string]any)["requests"].(map[string]any)["filter"] = "httpStatus:>=500"
			rule.Evaluation.Query, _ = json.Marshal(query)
		case http.MethodPatch:
			var patch map[string]json.RawMessage
			if err := json.NewDecoder(req.Body).Decode(&patch); err != nil {
				t.Error(err)
				http.Error(w, "invalid", 400)
				return
			}
			patches = append(patches, patch)
			existing, _ := json.Marshal(rule)
			var merged map[string]json.RawMessage
			_ = json.Unmarshal(existing, &merged)
			for key, value := range patch {
				merged[key] = value
			}
			existing, _ = json.Marshal(merged)
			rule = client.AlertRule{}
			_ = json.Unmarshal(existing, &rule)
			var query map[string]any
			_ = json.Unmarshal(rule.Evaluation.Query, &query)
			metric := query["metrics"].(map[string]any)["requests"].(map[string]any)
			metric["filter"] = strings.ReplaceAll(metric["filter"].(string), "httpStatus >= 500", "httpStatus:>=500")
			if concurrentQueryChange && len(patches) == 1 {
				metric["filter"] = "httpStatus:>=400"
			}
			rule.Evaluation.Query, _ = json.Marshal(query)
		case http.MethodGet:
			if deleted {
				http.Error(w, `{"error":{"code":"not_found","message":"missing"}}`, 404)
				return
			}
		case http.MethodDelete:
			deleted = true
			_, _ = fmt.Fprint(w, `{"success":true}`)
			return
		default:
			t.Errorf("unexpected method %s", req.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"rule": rule})
	}))
	t.Cleanup(server.Close)
	factories := map[string]func() (tfprotov6.ProviderServer, error){"vercel": providerserver.NewProtocol6WithError(&customAlertMockProvider{
		Provider: New(), client: client.New("TOKEN").WithBaseURL(server.URL).WithTeam(client.Team{ID: "team_123"}),
	})}
	config := func(name, extra, trigger string) string {
		return fmt.Sprintf(`
resource "vercel_custom_alert_rule" "test" {
 name = %q
 project_id = "prj_123"
 severity = "high"
 %s
 evaluation = {
  window = "5m"
  query = {metrics = {requests = {metric = "vercel.request.count", aggregation = "count", filter = "httpStatus >= 500"}}, outputs = ["requests"]}
 }
 trigger = %s
}
`, name, extra, trigger)
	}
	threshold := `{type = "threshold", output = "requests", operator = "gt", threshold = 0}`
	anomaly := `{type = "anomaly", output = "requests", standard_deviations = 3, minimum = {output = "requests", threshold = 10}}`
	steps := []tfresource.TestStep{
		{Config: config("Requests", "investigation_prompt = \"Investigate checkout failures\"\n tags = [\"checkout\"]", threshold), Check: tfresource.ComposeTestCheckFunc(
			tfresource.TestCheckResourceAttr("vercel_custom_alert_rule.test", "id", "ar_123"),
			tfresource.TestCheckResourceAttr("vercel_custom_alert_rule.test", "trigger.threshold", "0"),
			tfresource.TestCheckResourceAttr("vercel_custom_alert_rule.test", "team_id", "team_123"),
			tfresource.TestCheckResourceAttr("vercel_custom_alert_rule.test", "notification_settings.enable_team_owner_notifications", "true"),
			tfresource.TestCheckTypeSetElemAttr("vercel_custom_alert_rule.test", "tags.*", "checkout"),
		)},
		{Config: config("Renamed", "", threshold)},
		{Config: config("Renamed", "", anomaly)},
		{ResourceName: "vercel_custom_alert_rule.test", ImportState: true, ImportStateId: "team_123/ar_123", ImportStateVerify: true, ImportStateVerifyIgnore: []string{"evaluation.query"}},
	}
	expectedPatches := 2
	if concurrentQueryChange {
		failedUpdate := tfresource.TestStep{
			Config:      config("Renamed", "", threshold),
			ExpectError: regexp.MustCompile("Custom Alert Rule query changed during update"),
		}
		steps = append(steps[:1], append([]tfresource.TestStep{failedUpdate}, steps[1:]...)...)
		expectedPatches++
	}
	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: factories, Steps: steps})
	mu.Lock()
	defer mu.Unlock()
	if !deleted || len(patches) != expectedPatches {
		t.Fatalf("deleted = %v, patches = %#v", deleted, patches)
	}
	if len(patches[0]) != 3 || string(patches[0]["name"]) != `"Renamed"` || string(patches[0]["investigationPrompt"]) != "null" || string(patches[0]["tags"]) != "null" {
		t.Fatalf("metadata patch = %#v", patches[0])
	}
	if concurrentQueryChange {
		if len(patches[1]) != 1 || patches[1]["evaluation"] == nil {
			t.Fatalf("drift repair patch = %#v", patches[1])
		}
		var query map[string]any
		if err := json.Unmarshal(rule.Evaluation.Query, &query); err != nil {
			t.Fatal(err)
		}
		if query["metrics"].(map[string]any)["requests"].(map[string]any)["filter"] != "httpStatus:>=500" {
			t.Fatalf("query drift was not repaired: %s", rule.Evaluation.Query)
		}
	}
	triggerPatch := patches[len(patches)-1]
	if len(triggerPatch) != 1 || triggerPatch["trigger"] == nil {
		t.Fatalf("trigger patch = %#v", triggerPatch)
	}
}

func TestCustomAlertRuleReadAndDeleteFailures(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusForbidden, http.StatusInternalServerError} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(status)
				_, _ = fmt.Fprint(w, `{"error":{"code":"test_error","message":"mock failure"}}`)
			}))
			t.Cleanup(server.Close)
			ctx := context.Background()
			state := tfsdk.State{Schema: customAlertRuleSchema(t)}
			if diags := state.Set(ctx, customAlertTestModel()); diags.HasError() {
				t.Fatal(diags)
			}
			r := &customAlertRuleResource{client: client.New("TOKEN").WithBaseURL(server.URL)}
			read := resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, &read)
			if status == http.StatusNotFound {
				if read.Diagnostics.HasError() || !read.State.Raw.IsNull() {
					t.Fatalf("missing rule: diagnostics = %v, state = %v", read.Diagnostics, read.State.Raw)
				}
			} else if !read.Diagnostics.HasError() || !read.State.Raw.Equal(state.Raw) {
				t.Fatalf("failed read lost state: diagnostics = %v, state = %v", read.Diagnostics, read.State.Raw)
			}
			var deleted resource.DeleteResponse
			r.Delete(ctx, resource.DeleteRequest{State: state}, &deleted)
			if deleted.Diagnostics.HasError() != (status != http.StatusNotFound) {
				t.Fatalf("delete diagnostics = %v", deleted.Diagnostics)
			}
		})
	}
}

func TestCustomAlertRuleCreateRequiresTeam(t *testing.T) {
	ctx := context.Background()
	model := customAlertTestModel()
	model.TeamID = types.StringNull()
	plan := tfsdk.Plan{Schema: customAlertRuleSchema(t)}
	if diags := plan.Set(ctx, model); diags.HasError() {
		t.Fatal(diags)
	}
	var response resource.CreateResponse
	(&customAlertRuleResource{client: client.New("TOKEN")}).Create(ctx, resource.CreateRequest{Plan: plan}, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("creation without a team was accepted")
	}
}
