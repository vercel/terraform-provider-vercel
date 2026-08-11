package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateAlertRuleBuiltIn(t *testing.T) {
	filter := "statusGroup eq '5xx'"
	projectFilter := "projectId in ('prj_123')"
	sensitivity := int64(3)
	autosubscribe := false

	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/alerts/v2/alert-rules" {
			t.Fatalf("request = %s %s, want POST /alerts/v2/alert-rules", r.Method, r.URL.Path)
		}
		if teamID := r.URL.Query().Get("teamId"); teamID != "team_123" {
			t.Fatalf("teamId = %q, want team_123", teamID)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"ar_123","teamId":"team_123","name":"5xx anomalies",
			"projectId":"projectId in ('prj_123')",
			"alertTypes":[{"type":"error_anomaly","filter":"statusGroup eq '5xx'"}],
			"sensitivityLevel":3,
			"autosubscribeOwnersInKnock":false,
			"action":"trigger"
		}`))
	}))
	t.Cleanup(server.Close)

	alertRule, err := New("TOKEN").WithBaseURL(server.URL).CreateAlertRule(context.Background(), CreateAlertRuleRequest{
		TeamID:              "team_123",
		Name:                "5xx anomalies",
		ProjectID:           &projectFilter,
		AlertTypes:          []AlertRuleType{{Type: AlertRuleTypeErrorAnomaly, Filter: &filter}},
		SensitivityLevel:    &sensitivity,
		AutosubscribeOwners: &autosubscribe,
	})
	if err != nil {
		t.Fatalf("CreateAlertRule() error = %v", err)
	}

	// The create payload omits optional fields that were not set, so that the
	// API applies its own defaults rather than clearing them.
	if _, ok := got["autosubscribeProjectAdminsInKnock"]; ok {
		t.Fatalf("create payload should omit unset optional fields, got %#v", got)
	}
	if _, ok := got["customAlert"]; ok {
		t.Fatalf("create payload should omit customAlert for built-in rules, got %#v", got)
	}
	if got["projectId"] != projectFilter {
		t.Fatalf("projectId = %v, want %q", got["projectId"], projectFilter)
	}

	if alertRule.ID != "ar_123" || alertRule.TeamID != "team_123" {
		t.Fatalf("alertRule = %#v", alertRule)
	}
	if alertRule.IsCustomAlert() {
		t.Fatalf("IsCustomAlert() = true, want false for %#v", alertRule.AlertTypes)
	}
	if alertRule.SensitivityLevel == nil || *alertRule.SensitivityLevel != 3 {
		t.Fatalf("sensitivityLevel = %v, want 3", alertRule.SensitivityLevel)
	}
	if alertRule.AutosubscribeProjectAdmins != nil {
		t.Fatalf("autosubscribeProjectAdmins = %v, want nil", *alertRule.AutosubscribeProjectAdmins)
	}
	if len(alertRule.AlertTypes) != 1 || alertRule.AlertTypes[0].Filter == nil || *alertRule.AlertTypes[0].Filter != filter {
		t.Fatalf("alertTypes = %#v", alertRule.AlertTypes)
	}
}

func TestCreateAlertRuleCustomAlertDerivesQueryScope(t *testing.T) {
	projectID := "prj_123"
	rollupFilter := "httpStatus ge 500"
	minThreshold := 20.0
	hours := int64(1)

	var got struct {
		ProjectID   string             `json:"projectId"`
		CustomAlert customAlertPayload `json:"customAlert"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"ar_custom","teamId":"team_123","name":"Checkout error rate",
			"projectId":"prj_123",
			"alertTypes":[{"type":"custom_alert"}],
			"customAlert":{
				"queryJsonString":"{\"scope\":{\"type\":\"project\",\"ownerId\":\"team_123\",\"projectIds\":[\"prj_123\"]},\"event\":\"incomingRequest\",\"rollups\":{\"errors\":{\"measure\":\"count\",\"aggregation\":\"sum\",\"filter\":\"httpStatus ge 500\"},\"requests\":{\"measure\":\"count\",\"aggregation\":\"sum\"}},\"granularity\":{\"hours\":1}}",
				"triggerType":"threshold",
				"triggerOperator":"gt",
				"triggerThreshold":0.05,
				"minThreshold":20,
				"formula":{"operator":"divide","left":"errors","right":"requests"}
			}
		}`))
	}))
	t.Cleanup(server.Close)

	alertRule, err := New("TOKEN").WithBaseURL(server.URL).CreateAlertRule(context.Background(), CreateAlertRuleRequest{
		TeamID:     "team_123",
		Name:       "Checkout error rate",
		ProjectID:  &projectID,
		AlertTypes: []AlertRuleType{{Type: AlertRuleTypeCustomAlert}},
		CustomAlert: &CustomAlert{
			Query: CustomAlertQuery{
				Event: "incomingRequest",
				Rollups: map[string]CustomAlertRollup{
					"errors":   {Measure: "count", Aggregation: "sum", Filter: &rollupFilter},
					"requests": {Measure: "count", Aggregation: "sum"},
				},
				Granularity: &CustomAlertGranularity{Hours: &hours},
			},
			TriggerType:      "threshold",
			TriggerOperator:  "gt",
			TriggerThreshold: 0.05,
			MinThreshold:     &minThreshold,
			Formula:          &CustomAlertFormula{Operator: "divide", Left: "errors", Right: "requests"},
		},
	})
	if err != nil {
		t.Fatalf("CreateAlertRule() error = %v", err)
	}

	// The API rejects custom alert queries that have no scope, so the client
	// derives it from the rule's team and project.
	var sentQuery CustomAlertQuery
	if err := json.Unmarshal([]byte(got.CustomAlert.QueryJSONString), &sentQuery); err != nil {
		t.Fatalf("Unmarshal(queryJsonString) error = %v", err)
	}
	if sentQuery.Scope == nil {
		t.Fatalf("query scope = nil, want a derived project scope")
	}
	if sentQuery.Scope.Type != "project" || sentQuery.Scope.OwnerID != "team_123" {
		t.Fatalf("query scope = %#v", sentQuery.Scope)
	}
	if len(sentQuery.Scope.ProjectIDs) != 1 || sentQuery.Scope.ProjectIDs[0] != projectID {
		t.Fatalf("query scope projectIds = %#v, want [%q]", sentQuery.Scope.ProjectIDs, projectID)
	}
	if got.ProjectID != projectID {
		t.Fatalf("projectId = %q, want %q (custom alerts use a raw project ID)", got.ProjectID, projectID)
	}

	if !alertRule.IsCustomAlert() {
		t.Fatalf("IsCustomAlert() = false, want true")
	}
	if alertRule.CustomAlert == nil {
		t.Fatalf("customAlert = nil")
	}
	if alertRule.CustomAlert.Query.Event != "incomingRequest" {
		t.Fatalf("query event = %q", alertRule.CustomAlert.Query.Event)
	}
	if len(alertRule.CustomAlert.Query.Rollups) != 2 {
		t.Fatalf("query rollups = %#v", alertRule.CustomAlert.Query.Rollups)
	}
	errors := alertRule.CustomAlert.Query.Rollups["errors"]
	if errors.Filter == nil || *errors.Filter != rollupFilter {
		t.Fatalf("errors rollup = %#v", errors)
	}
	if alertRule.CustomAlert.Query.Granularity == nil || alertRule.CustomAlert.Query.Granularity.Hours == nil {
		t.Fatalf("granularity = %#v", alertRule.CustomAlert.Query.Granularity)
	}
	if alertRule.CustomAlert.Formula == nil || alertRule.CustomAlert.Formula.Left != "errors" {
		t.Fatalf("formula = %#v", alertRule.CustomAlert.Formula)
	}
	if alertRule.CustomAlert.MinThreshold == nil || *alertRule.CustomAlert.MinThreshold != 20 {
		t.Fatalf("minThreshold = %v", alertRule.CustomAlert.MinThreshold)
	}
}

func TestCreateAlertRuleCustomAlertRequiresProject(t *testing.T) {
	_, err := New("TOKEN").CreateAlertRule(context.Background(), CreateAlertRuleRequest{
		TeamID:     "team_123",
		Name:       "no project",
		AlertTypes: []AlertRuleType{{Type: AlertRuleTypeCustomAlert}},
		CustomAlert: &CustomAlert{
			Query:            CustomAlertQuery{Event: "incomingRequest"},
			TriggerType:      "threshold",
			TriggerOperator:  "gt",
			TriggerThreshold: 1,
		},
	})
	if err == nil {
		t.Fatalf("CreateAlertRule() error = nil, want an error when no project is targeted")
	}
}

func TestUpdateAlertRuleClearsOmittedFields(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/alerts/v2/alert-rules/ar_123" {
			t.Fatalf("request = %s %s, want PATCH /alerts/v2/alert-rules/ar_123", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"ar_123","teamId":"team_123","name":"renamed",
			"alertTypes":[{"type":"usage_anomaly"}],
			"action":"trigger"
		}`))
	}))
	t.Cleanup(server.Close)

	alertRule, err := New("TOKEN").WithBaseURL(server.URL).UpdateAlertRule(context.Background(), UpdateAlertRuleRequest{
		TeamID:     "team_123",
		ID:         "ar_123",
		Name:       "renamed",
		AlertTypes: []AlertRuleType{{Type: AlertRuleTypeUsageAnomaly}},
	})
	if err != nil {
		t.Fatalf("UpdateAlertRule() error = %v", err)
	}

	// The API clears optional fields when they are sent as explicit nulls, so
	// the update payload has to include them.
	for _, field := range []string{"projectId", "sensitivityLevel", "autosubscribeOwnersInKnock", "autosubscribeProjectAdminsInKnock"} {
		value, ok := got[field]
		if !ok {
			t.Fatalf("update payload is missing %q, which is how the API clears it: %#v", field, got)
		}
		if value != nil {
			t.Fatalf("update payload %q = %v, want null", field, value)
		}
	}
	// The API rejects a null customAlert, so it has to be omitted instead.
	if _, ok := got["customAlert"]; ok {
		t.Fatalf("update payload should omit customAlert when unset, got %#v", got)
	}

	if alertRule.ProjectID != nil || alertRule.SensitivityLevel != nil || alertRule.AutosubscribeOwners != nil {
		t.Fatalf("alertRule = %#v", alertRule)
	}
}

func TestGetAlertRuleNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"Alert Rule not found."}}`))
	}))
	t.Cleanup(server.Close)

	_, err := New("TOKEN").WithBaseURL(server.URL).GetAlertRule(context.Background(), "ar_missing", "team_123")
	if !NotFound(err) {
		t.Fatalf("NotFound(%v) = false, want true", err)
	}
}

func TestListAlertRulesFiltersByProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("projectId") != "prj_123" {
			t.Fatalf("projectId = %q, want prj_123", r.URL.Query().Get("projectId"))
		}
		if r.URL.Query().Get("teamId") != "team_123" {
			t.Fatalf("teamId = %q, want team_123", r.URL.Query().Get("teamId"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":"ar_default","teamId":"team_123","name":"Default Alert Rule","isDefault":true},
			{"id":"ar_123","teamId":"team_123","name":"5xx","alertTypes":[{"type":"error_anomaly"}]}
		]`))
	}))
	t.Cleanup(server.Close)

	alertRules, err := New("TOKEN").WithBaseURL(server.URL).ListAlertRules(context.Background(), "team_123", "prj_123")
	if err != nil {
		t.Fatalf("ListAlertRules() error = %v", err)
	}
	if len(alertRules) != 2 {
		t.Fatalf("alertRules = %#v, want 2", alertRules)
	}
	if !alertRules[0].IsDefault {
		t.Fatalf("alertRules[0].IsDefault = false, want true")
	}
}

func TestAlertRuleFromResponseInvalidQueryJSON(t *testing.T) {
	_, err := alertRuleFromResponse(alertRuleResponse{
		ID:          "ar_123",
		CustomAlert: &customAlertPayload{QueryJSONString: "not json"},
	})
	if err == nil {
		t.Fatalf("alertRuleFromResponse() error = nil, want a decoding error")
	}
}

func TestCustomAlertToPayloadKeepsExplicitScope(t *testing.T) {
	payload, err := customAlertToPayload(&CustomAlert{
		Query: CustomAlertQuery{
			Scope: &CustomAlertQueryScope{Type: "project", OwnerID: "team_explicit", ProjectIDs: []string{"prj_explicit"}},
			Event: "incomingRequest",
		},
	}, "team_123", nil)
	if err != nil {
		t.Fatalf("customAlertToPayload() error = %v", err)
	}

	var query CustomAlertQuery
	if err := json.Unmarshal([]byte(payload.QueryJSONString), &query); err != nil {
		t.Fatalf("Unmarshal(queryJsonString) error = %v", err)
	}
	if query.Scope.OwnerID != "team_explicit" || query.Scope.ProjectIDs[0] != "prj_explicit" {
		t.Fatalf("query scope = %#v, want the explicitly configured scope", query.Scope)
	}
}
