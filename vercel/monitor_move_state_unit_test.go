package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func moveNotificationState(t *testing.T, source resource.Resource, target resource.Resource, sourceType string, model any) (resource.MoveStateResponse, tfsdk.State) {
	t.Helper()
	ctx := context.Background()
	sourceState := tfsdk.State{Schema: *schemaOf(ctx, source)}
	if diags := sourceState.Set(ctx, model); diags.HasError() {
		t.Fatal(diags)
	}
	movers := target.(resource.ResourceWithMoveState).MoveState(ctx)
	if len(movers) != 1 {
		t.Fatalf("got %d state movers, want 1", len(movers))
	}
	resp := resource.MoveStateResponse{TargetState: tfsdk.State{Schema: *schemaOf(ctx, target)}}
	movers[0].StateMover(ctx, resource.MoveStateRequest{
		SourceState:           &sourceState,
		SourceTypeName:        sourceType,
		SourceProviderAddress: "registry.terraform.io/vercel/vercel",
	}, &resp)
	return resp, sourceState
}

func TestMonitorNotificationsMoveFromAlertRuleNotifications(t *testing.T) {
	ctx := context.Background()

	resp, _ := moveNotificationState(t, &alertRuleSlackNotificationResource{}, &monitorSlackNotificationResource{}, "vercel_alert_rule_slack_notification", AlertRuleSlackNotification{
		ID: types.StringValue("team_1/ar_1/icfg_1/C1"), TeamID: types.StringValue("team_1"), AlertRuleID: types.StringValue("ar_1"),
		SlackChannelID: types.StringValue("C1"), SlackInstallationID: types.StringValue("icfg_1"), MinimumSeverityLevel: types.StringValue("critical"),
	})
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var slack MonitorSlackNotification
	if diags := resp.TargetState.Get(ctx, &slack); diags.HasError() {
		t.Fatal(diags)
	}
	if slack.MonitorID.ValueString() != "ar_1" || slack.SlackChannelID.ValueString() != "C1" || slack.SlackInstallationID.ValueString() != "icfg_1" || slack.MinimumSeverityLevel.ValueString() != "critical" || slack.ID.ValueString() != "team_1/ar_1/icfg_1/C1" {
		t.Fatalf("moved slack state = %#v", slack)
	}

	resp, _ = moveNotificationState(t, &alertRuleWebhookNotificationResource{}, &monitorWebhookNotificationResource{}, "vercel_alert_rule_webhook_notification", AlertRuleWebhookNotification{
		ID: types.StringValue("team_1/ar_1/hook_1"), TeamID: types.StringValue("team_1"), AlertRuleID: types.StringValue("ar_1"),
		WebhookID: types.StringValue("hook_1"), MinimumSeverityLevel: types.StringValue("high"),
	})
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var webhook MonitorWebhookNotification
	if diags := resp.TargetState.Get(ctx, &webhook); diags.HasError() {
		t.Fatal(diags)
	}
	if webhook.MonitorID.ValueString() != "ar_1" || webhook.WebhookID.ValueString() != "hook_1" || webhook.MinimumSeverityLevel.ValueString() != "high" {
		t.Fatalf("moved webhook state = %#v", webhook)
	}
}

func TestCustomMonitorMovesFromCustomAlertRule(t *testing.T) {
	ctx := context.Background()
	model := customAlertTestModel(t)
	resp, _ := moveNotificationState(t, &customAlertRuleResource{}, &customMonitorResource{}, "vercel_custom_alert_rule", model)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var moved CustomAlertRule
	if diags := resp.TargetState.Get(ctx, &moved); diags.HasError() {
		t.Fatal(diags)
	}
	if moved.ID.ValueString() != "ar_123" || moved.ProjectID.ValueString() != "prj_123" || !moved.Evaluation.Equal(model.Evaluation) {
		t.Fatalf("moved custom state = %#v", moved)
	}
}

func TestMonitorMoveIgnoresOtherSources(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name, typeName, address string
	}{
		{"other type", "vercel_project", "registry.terraform.io/vercel/vercel"},
		{"other provider", "vercel_alert_rule", "registry.terraform.io/hashicorp/random"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			movers := (&monitorResource{}).MoveState(ctx)
			resp := resource.MoveStateResponse{TargetState: tfsdk.State{Schema: *schemaOf(ctx, &monitorResource{})}}
			movers[0].StateMover(ctx, resource.MoveStateRequest{SourceTypeName: tc.typeName, SourceProviderAddress: tc.address}, &resp)
			if resp.Diagnostics.HasError() || !resp.TargetState.Raw.IsNull() {
				t.Fatalf("mover handled the request: %v", resp.Diagnostics)
			}
		})
	}
}

// A `moved` block must adopt the existing alert rule as a monitor without
// recreating it or changing the plan afterwards, even when the API
// canonicalizes the configured filter.
func TestMonitorMovedBlockAdoptsAlertRule(t *testing.T) {
	var mu sync.Mutex
	var rule client.AlertRule
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		requests = append(requests, req.Method+" "+req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		envelope := "rule"
		if strings.HasPrefix(req.URL.Path, "/alerts/v3/monitors") {
			envelope = "monitor"
		}
		switch req.Method {
		case http.MethodPost:
			if err := json.NewDecoder(req.Body).Decode(&rule); err != nil {
				t.Error(err)
			}
			rule.ID = "ar_123"
			// Simulate the API's KQL canonicalization.
			for i := range rule.Triggers.Items {
				rule.Triggers.Items[i].Filter = strPtr("statusGroup:(5xx)")
			}
		case http.MethodPatch:
			t.Errorf("unexpected update %s", req.URL.Path)
		case http.MethodDelete:
			_, _ = fmt.Fprint(w, `{"success":true}`)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{envelope: rule})
	}))
	t.Cleanup(server.Close)
	factories := map[string]func() (tfprotov6.ProviderServer, error){"vercel": providerserver.NewProtocol6WithError(&customAlertMockProvider{
		Provider: New(), client: client.New("TOKEN").WithBaseURL(server.URL).WithTeam(client.Team{ID: "team_123"}),
	})}
	body := `
  type = "built-in"
  name = "Errors"
  rule_scope = { type = "all" }
  triggers = [{ type = "error_anomaly", filter = "statusGroup:5xx" }]
  match_minimum_severity_level = "high"
`
	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: factories, Steps: []tfresource.TestStep{
		{Config: `resource "vercel_alert_rule" "errors" {` + body + `}`},
		{Config: `
moved {
  from = vercel_alert_rule.errors
  to   = vercel_monitor.errors
}

resource "vercel_monitor" "errors" {` + body + `}`, Check: tfresource.TestCheckResourceAttr("vercel_monitor.errors", "id", "ar_123")},
	}})
	mu.Lock()
	defer mu.Unlock()
	for _, request := range requests {
		if request == "POST /alerts/v3/monitors" {
			t.Fatalf("the monitor was recreated: %v", requests)
		}
	}
	if len(requests) == 0 || requests[len(requests)-1] != "DELETE /alerts/v3/monitors/ar_123" {
		t.Fatalf("requests = %v, want the monitor destroyed through the monitors API", requests)
	}
}

func TestCustomMonitorMovedBlockAdoptsCustomAlertRule(t *testing.T) {
	var mu sync.Mutex
	var rule client.AlertRule
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		requests = append(requests, req.Method+" "+req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		envelope := "rule"
		if strings.HasPrefix(req.URL.Path, "/alerts/v3/monitors") {
			envelope = "monitor"
		}
		switch req.Method {
		case http.MethodPost:
			if err := json.NewDecoder(req.Body).Decode(&rule); err != nil {
				t.Error(err)
			}
			rule.ID, rule.QuerySupported = "ar_123", true
			// Simulate the API's KQL canonicalization.
			var query map[string]any
			_ = json.Unmarshal(rule.Evaluation.Query, &query)
			query["metrics"].(map[string]any)["requests"].(map[string]any)["filter"] = "httpStatus:>=500"
			rule.Evaluation.Query, _ = json.Marshal(query)
		case http.MethodPatch:
			t.Errorf("unexpected update %s", req.URL.Path)
		case http.MethodDelete:
			_, _ = fmt.Fprint(w, `{"success":true}`)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{envelope: rule})
	}))
	t.Cleanup(server.Close)
	factories := map[string]func() (tfprotov6.ProviderServer, error){"vercel": providerserver.NewProtocol6WithError(&customAlertMockProvider{
		Provider: New(), client: client.New("TOKEN").WithBaseURL(server.URL).WithTeam(client.Team{ID: "team_123"}),
	})}
	body := `
  name = "Requests"
  project_id = "prj_123"
  severity = "high"
  evaluation = {
    window = "5m"
    query = {metrics = {requests = {metric = "vercel.request.count", aggregation = "count", filter = "httpStatus >= 500"}}, outputs = ["requests"]}
  }
  trigger = {type = "threshold", output = "requests", operator = "gt", threshold = 0}
`
	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: factories, Steps: []tfresource.TestStep{
		{Config: `resource "vercel_custom_alert_rule" "requests" {` + body + `}`},
		{Config: `
moved {
  from = vercel_custom_alert_rule.requests
  to   = vercel_custom_monitor.requests
}

resource "vercel_custom_monitor" "requests" {` + body + `}`, Check: tfresource.TestCheckResourceAttr("vercel_custom_monitor.requests", "id", "ar_123")},
	}})
	mu.Lock()
	defer mu.Unlock()
	for _, request := range requests {
		if request == "POST /alerts/v3/monitors" || strings.HasPrefix(request, "PATCH") {
			t.Fatalf("the monitor was recreated or updated: %v", requests)
		}
	}
}
