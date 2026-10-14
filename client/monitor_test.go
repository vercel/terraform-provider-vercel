package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateMonitor(t *testing.T) {
	filter := "statusGroup:5xx"
	severity := "high"

	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/alerts/v3/monitors" {
			t.Fatalf("request = %s %s, want POST /alerts/v3/monitors", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("teamId"); got != "team_123" {
			t.Fatalf("teamId = %q, want team_123", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"monitor":{"id":"ar_123","type":"built-in","name":"5xx anomalies","ruleScope":{"type":"all"},"triggers":{"mode":"selected","items":[{"type":"error_anomaly","filter":"statusGroup:5xx"}]},"matchMinimumSeverityLevel":"high","notificationSettings":{"enableTeamOwnerNotifications":true},"isDefault":false}}`))
	}))
	t.Cleanup(server.Close)

	monitor, err := New("TOKEN").WithBaseURL(server.URL).CreateMonitor(context.Background(), CreateAlertRuleRequest{
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
		t.Fatalf("CreateMonitor() error = %v", err)
	}
	if _, ok := body["teamId"]; ok {
		t.Fatalf("request body contains teamId: %#v", body)
	}
	if body["type"] != AlertRuleTypeBuiltIn || body["name"] != "5xx anomalies" {
		t.Fatalf("request body = %#v", body)
	}
	if monitor.ID != "ar_123" || monitor.Triggers == nil || len(monitor.Triggers.Items) != 1 {
		t.Fatalf("monitor = %#v", monitor)
	}
}

func TestGetMonitor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/alerts/v3/monitors/ar_123" {
			t.Fatalf("request = %s %s, want GET /alerts/v3/monitors/ar_123", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"monitor":{"id":"ar_123","type":"built-in","name":"one","ruleScope":{"type":"all"},"notificationSettings":{"enableTeamOwnerNotifications":true},"isDefault":false}}`))
	}))
	t.Cleanup(server.Close)

	monitor, err := New("TOKEN").WithBaseURL(server.URL).GetMonitor(context.Background(), "ar_123", "team_123")
	if err != nil {
		t.Fatalf("GetMonitor() error = %v", err)
	}
	if monitor.ID != "ar_123" || monitor.Name != "one" {
		t.Fatalf("monitor = %#v", monitor)
	}
}

func TestUpdateMonitor(t *testing.T) {
	severity := "medium"

	var body UpdateAlertRuleRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/alerts/v3/monitors/ar_123" {
			t.Fatalf("request = %s %s, want PATCH /alerts/v3/monitors/ar_123", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"monitor":{"id":"ar_123","type":"built-in","name":"5xx anomalies","ruleScope":{"type":"all"},"matchMinimumSeverityLevel":"medium","notificationSettings":{"enableTeamOwnerNotifications":true},"isDefault":false}}`))
	}))
	t.Cleanup(server.Close)

	monitor, err := New("TOKEN").WithBaseURL(server.URL).UpdateMonitor(context.Background(), UpdateAlertRuleRequest{
		TeamID:                    "team_123",
		ID:                        "ar_123",
		MatchMinimumSeverityLevel: &severity,
	})
	if err != nil {
		t.Fatalf("UpdateMonitor() error = %v", err)
	}
	if body.TeamID != "" || body.ID != "" {
		t.Fatalf("decoded transport-only fields = %#v", body)
	}
	if body.MatchMinimumSeverityLevel == nil || *body.MatchMinimumSeverityLevel != severity {
		t.Fatalf("request body = %#v", body)
	}
	if monitor.MatchMinimumSeverityLevel == nil || *monitor.MatchMinimumSeverityLevel != severity {
		t.Fatalf("monitor = %#v", monitor)
	}
}

func TestDeleteMonitor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/alerts/v3/monitors/ar_123" {
			t.Fatalf("request = %s %s, want DELETE /alerts/v3/monitors/ar_123", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	if err := New("TOKEN").WithBaseURL(server.URL).DeleteMonitor(context.Background(), "ar_123", "team_123"); err != nil {
		t.Fatalf("DeleteMonitor() error = %v", err)
	}
}

func TestListMonitorsPaginates(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/alerts/v3/monitors" {
			t.Fatalf("path = %q, want /alerts/v3/monitors", r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "100" {
			t.Fatalf("limit = %q, want 100", got)
		}
		w.Header().Set("Content-Type", "application/json")
		if requests == 1 {
			if got := r.URL.Query().Get("cursor"); got != "" {
				t.Fatalf("first cursor = %q, want empty", got)
			}
			_, _ = w.Write([]byte(`{"monitors":[{"id":"ar_1","type":"built-in","name":"one","ruleScope":{"type":"all"},"notificationSettings":{"enableTeamOwnerNotifications":true},"isDefault":false}],"pagination":{"next":"next_cursor"}}`))
			return
		}
		if got := r.URL.Query().Get("cursor"); got != "next_cursor" {
			t.Fatalf("second cursor = %q, want next_cursor", got)
		}
		_, _ = w.Write([]byte(`{"monitors":[{"id":"ar_2","type":"built-in","name":"two","ruleScope":{"type":"all"},"notificationSettings":{"enableTeamOwnerNotifications":true},"isDefault":false}],"pagination":{"next":null}}`))
	}))
	t.Cleanup(server.Close)

	monitors, err := New("TOKEN").WithBaseURL(server.URL).ListMonitors(context.Background(), "team_123")
	if err != nil {
		t.Fatalf("ListMonitors() error = %v", err)
	}
	if len(monitors) != 2 || requests != 2 || monitors[1].ID != "ar_2" {
		t.Fatalf("monitors = %#v, requests = %d", monitors, requests)
	}
}
