package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateAlertRule(t *testing.T) {
	filter := "statusGroup:5xx"
	severity := "high"

	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/alerts/v3/alert-rules" {
			t.Fatalf("request = %s %s, want POST /alerts/v3/alert-rules", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("teamId"); got != "team_123" {
			t.Fatalf("teamId = %q, want team_123", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"rule":{"id":"ar_123","type":"built-in","name":"5xx anomalies","ruleScope":{"type":"all"},"triggers":{"mode":"selected","items":[{"type":"error_anomaly","filter":"statusGroup:5xx"}]},"matchMinimumSeverityLevel":"high","notificationSettings":{"enableTeamOwnerNotifications":true},"isDefault":false}}`))
	}))
	t.Cleanup(server.Close)

	rule, err := New("TOKEN").WithBaseURL(server.URL).CreateAlertRule(context.Background(), CreateAlertRuleRequest{
		TeamID: "team_123",
		AlertRuleCreate: AlertRuleCreate{
			Type:                      AlertRuleTypeBuiltIn,
			Name:                      "5xx anomalies",
			RuleScope:                 AlertRuleScope{Type: "all"},
			Triggers:                  &AlertRuleTriggers{Mode: "selected", Items: []AlertRuleTrigger{{Type: "error_anomaly", Filter: &filter}}},
			MatchMinimumSeverityLevel: &severity,
		},
	})
	if err != nil {
		t.Fatalf("CreateAlertRule() error = %v", err)
	}
	if _, ok := body["teamId"]; ok {
		t.Fatalf("request body contains teamId: %#v", body)
	}
	if body["type"] != AlertRuleTypeBuiltIn || body["name"] != "5xx anomalies" {
		t.Fatalf("request body = %#v", body)
	}
	if rule.ID != "ar_123" || rule.Triggers == nil || len(rule.Triggers.Items) != 1 {
		t.Fatalf("rule = %#v", rule)
	}
}

func TestUpdateCustomAlertRule(t *testing.T) {
	operator := "gte"
	threshold := 10.0
	querySupported := true

	var body UpdateAlertRuleRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/alerts/v3/alert-rules/ar_custom" {
			t.Fatalf("request = %s %s, want PATCH /alerts/v3/alert-rules/ar_custom", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"rule":{"id":"ar_custom","type":"custom","name":"Request count","ruleScope":{"type":"project","projectId":"prj_123"},"severity":"medium","evaluation":{"window":"5m","query":{"metrics":{"requests":{"metric":"vercel.request.count","aggregation":"sum"}},"outputs":["requests"]}},"trigger":{"type":"threshold","output":"requests","operator":"gte","threshold":10},"notificationSettings":{"enableTeamOwnerNotifications":false},"isDefault":false,"querySupported":true}}`))
	}))
	t.Cleanup(server.Close)

	rule, err := New("TOKEN").WithBaseURL(server.URL).UpdateAlertRule(context.Background(), UpdateAlertRuleRequest{
		TeamID: "team_123",
		ID:     "ar_custom",
		Name:   pointerTo("Request count"),
		Evaluation: &AlertRuleEvaluation{Window: "5m", Query: AlertRuleCustomQuery{
			Metrics: map[string]AlertRuleMetricSelection{"requests": {Metric: "vercel.request.count", Aggregation: "sum"}},
			Outputs: []string{"requests"},
		}},
		Trigger: &AlertRuleCustomTrigger{Type: "threshold", Output: "requests", Operator: &operator, Threshold: &threshold},
	})
	if err != nil {
		t.Fatalf("UpdateAlertRule() error = %v", err)
	}
	if body.TeamID != "" || body.ID != "" {
		t.Fatalf("decoded transport-only fields = %#v", body)
	}
	if body.Evaluation == nil || body.Trigger == nil || body.Trigger.Threshold == nil || *body.Trigger.Threshold != 10 {
		t.Fatalf("request body = %#v", body)
	}
	if rule.QuerySupported == nil || *rule.QuerySupported != querySupported || rule.Evaluation == nil {
		t.Fatalf("rule = %#v", rule)
	}
}

func TestListAlertRulesPaginates(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if got := r.URL.Query().Get("limit"); got != "100" {
			t.Fatalf("limit = %q, want 100", got)
		}
		w.Header().Set("Content-Type", "application/json")
		if requests == 1 {
			if got := r.URL.Query().Get("cursor"); got != "" {
				t.Fatalf("first cursor = %q, want empty", got)
			}
			_, _ = w.Write([]byte(`{"rules":[{"id":"ar_1","type":"built-in","name":"one","ruleScope":{"type":"all"},"triggers":{"mode":"selected","items":[]},"matchMinimumSeverityLevel":"low","notificationSettings":{"enableTeamOwnerNotifications":true},"isDefault":false}],"pagination":{"count":1,"next":"next_cursor"}}`))
			return
		}
		if got := r.URL.Query().Get("cursor"); got != "next_cursor" {
			t.Fatalf("second cursor = %q, want next_cursor", got)
		}
		_, _ = w.Write([]byte(`{"rules":[{"id":"ar_2","type":"built-in","name":"two","ruleScope":{"type":"all"},"triggers":{"mode":"selected","items":[]},"matchMinimumSeverityLevel":"low","notificationSettings":{"enableTeamOwnerNotifications":true},"isDefault":false}],"pagination":{"count":1,"next":null}}`))
	}))
	t.Cleanup(server.Close)

	rules, err := New("TOKEN").WithBaseURL(server.URL).ListAlertRules(context.Background(), "team_123")
	if err != nil {
		t.Fatalf("ListAlertRules() error = %v", err)
	}
	if len(rules) != 2 || requests != 2 || rules[1].ID != "ar_2" {
		t.Fatalf("rules = %#v, requests = %d", rules, requests)
	}
}

func pointerTo[T any](value T) *T {
	return &value
}
