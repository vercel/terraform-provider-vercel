package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const aiGatewayAPIKeyResponse = `{
	"apiKey": {
		"id": "key_123",
		"name": "workflow-github-actions",
		"partialKey": "abc",
		"teamId": "team_123",
		"purpose": "ai-gateway",
		"projectId": null,
		"expiresAt": null,
		"createdAt": 1700000000000
	}
}`

func TestCreateAIGatewayAPIKey(t *testing.T) {
	var got map[string]json.RawMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/api-keys" {
			t.Fatalf("request = %s %s, want POST /v1/api-keys", r.Method, r.URL.Path)
		}
		if teamID := r.URL.Query().Get("teamId"); teamID != "team_123" {
			t.Fatalf("teamId = %q, want team_123", teamID)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		_, _ = w.Write([]byte(`{
			"apiKeyString": "vck_secret",
			"apiKey": {
				"id": "key_123",
				"name": "workflow-github-actions",
				"partialKey": "abc",
				"teamId": "team_123",
				"purpose": "ai-gateway",
				"projectId": "prj_123",
				"expiresAt": 1800000000000,
				"createdAt": 1700000000000
			}
		}`))
	}))
	t.Cleanup(server.Close)

	expiresAt := int64(1800000000000)
	projectID := "prj_123"
	key, err := New("TOKEN").WithBaseURL(server.URL).CreateAIGatewayAPIKey(context.Background(), CreateAIGatewayAPIKeyRequest{
		Name:      "workflow-github-actions",
		Purpose:   "ai-gateway",
		ExpiresAt: &expiresAt,
		ProjectID: &projectID,
		AIGatewayQuota: &AIGatewayAPIKeyQuota{
			LimitAmount:     500,
			RefreshPeriod:   "monthly",
			AlertThresholds: []int64{75, 100},
		},
		TeamID: "team_123",
	})
	if err != nil {
		t.Fatalf("CreateAIGatewayAPIKey() error = %v", err)
	}
	if _, ok := got["teamId"]; ok {
		t.Fatalf("create request unexpectedly contains teamId: %s", got["teamId"])
	}
	if string(got["name"]) != `"workflow-github-actions"` || string(got["purpose"]) != `"ai-gateway"` {
		t.Fatalf("create request = %v", got)
	}
	if string(got["expiresAt"]) != `1800000000000` || string(got["projectId"]) != `"prj_123"` {
		t.Fatalf("create request scope = %v", got)
	}
	if string(got["aiGatewayQuota"]) != `{"limitAmount":500,"refreshPeriod":"monthly","alertThresholds":[75,100]}` {
		t.Fatalf("create request quota = %s", got["aiGatewayQuota"])
	}
	assertAIGatewayAPIKeyResponse(t, key)
	if key.ProjectID == nil || *key.ProjectID != "prj_123" || key.ExpiresAt == nil || *key.ExpiresAt != 1800000000000 {
		t.Fatalf("API key scope = %#v", key)
	}
	if key.APIKeyString == nil || *key.APIKeyString != "vck_secret" {
		t.Fatalf("APIKeyString = %#v, want vck_secret", key.APIKeyString)
	}
}

func TestGetAIGatewayAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/api-keys/key_123" {
			t.Fatalf("request = %s %s, want GET /v1/api-keys/key_123", r.Method, r.URL.Path)
		}
		if teamID := r.URL.Query().Get("teamId"); teamID != "team_123" {
			t.Fatalf("teamId = %q, want team_123", teamID)
		}
		_, _ = w.Write([]byte(aiGatewayAPIKeyResponse))
	}))
	t.Cleanup(server.Close)

	key, err := New("TOKEN").WithBaseURL(server.URL).GetAIGatewayAPIKey(context.Background(), "key_123", "team_123")
	if err != nil {
		t.Fatalf("GetAIGatewayAPIKey() error = %v", err)
	}
	assertAIGatewayAPIKeyResponse(t, key)
	if key.ProjectID != nil || key.ExpiresAt != nil || key.Quota != nil {
		t.Fatalf("API key scope = %#v", key)
	}
	if key.APIKeyString != nil {
		t.Fatalf("APIKeyString = %#v, want nil", key.APIKeyString)
	}
}

func TestGetAIGatewayAPIKeyQuotaPaginates(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/api-keys" {
			t.Fatalf("request = %s %s, want GET /v1/api-keys", r.Method, r.URL.Path)
		}
		if purpose := r.URL.Query().Get("purpose"); purpose != "ai-gateway" {
			t.Fatalf("purpose = %q, want ai-gateway", purpose)
		}
		if teamID := r.URL.Query().Get("teamId"); teamID != "team_123" {
			t.Fatalf("teamId = %q, want team_123", teamID)
		}
		requests++
		switch requests {
		case 1:
			if until := r.URL.Query().Get("until"); until != "" {
				t.Fatalf("until = %q, want empty on first page", until)
			}
			_, _ = w.Write([]byte(`{
				"apiKeys": [{"id": "key_other", "quota": {"limitAmount": 1, "refreshPeriod": "none"}}],
				"pagination": {"count": 1, "next": "cursor_2", "prev": null}
			}`))
		case 2:
			if until := r.URL.Query().Get("until"); until != "cursor_2" {
				t.Fatalf("until = %q, want cursor_2", until)
			}
			_, _ = w.Write([]byte(`{
				"apiKeys": [{"id": "key_123", "quota": {"limitAmount": 500, "refreshPeriod": "monthly", "alertThresholds": [75, 100]}}],
				"pagination": {"count": 1, "next": null, "prev": "cursor_1"}
			}`))
		default:
			t.Fatalf("unexpected request %d", requests)
		}
	}))
	t.Cleanup(server.Close)

	quota, err := New("TOKEN").WithBaseURL(server.URL).GetAIGatewayAPIKeyQuota(context.Background(), "key_123", "team_123")
	if err != nil {
		t.Fatalf("GetAIGatewayAPIKeyQuota() error = %v", err)
	}
	if quota == nil || quota.LimitAmount != 500 || quota.RefreshPeriod != "monthly" {
		t.Fatalf("quota = %#v", quota)
	}
	if len(quota.AlertThresholds) != 2 || quota.AlertThresholds[0] != 75 || quota.AlertThresholds[1] != 100 {
		t.Fatalf("alertThresholds = %#v", quota.AlertThresholds)
	}
}

func TestGetAIGatewayAPIKeyQuotaNoQuota(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"apiKeys": [{"id": "key_123"}],
			"pagination": {"count": 1, "next": null, "prev": null}
		}`))
	}))
	t.Cleanup(server.Close)

	quota, err := New("TOKEN").WithBaseURL(server.URL).GetAIGatewayAPIKeyQuota(context.Background(), "key_123", "team_123")
	if err != nil {
		t.Fatalf("GetAIGatewayAPIKeyQuota() error = %v", err)
	}
	if quota != nil {
		t.Fatalf("quota = %#v, want nil", quota)
	}
}

func TestUpdateAIGatewayAPIKey(t *testing.T) {
	var got map[string]json.RawMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/v1/api-keys/key_123" {
			t.Fatalf("request = %s %s, want PATCH /v1/api-keys/key_123", r.Method, r.URL.Path)
		}
		if teamID := r.URL.Query().Get("teamId"); teamID != "team_123" {
			t.Fatalf("teamId = %q, want team_123", teamID)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		_, _ = w.Write([]byte(`{
			"apiKey": {
				"id": "key_123",
				"name": "renamed-key",
				"partialKey": "abc",
				"teamId": "team_123",
				"purpose": "ai-gateway",
				"createdAt": 1700000000000
			}
		}`))
	}))
	t.Cleanup(server.Close)

	key, err := New("TOKEN").WithBaseURL(server.URL).UpdateAIGatewayAPIKey(context.Background(), UpdateAIGatewayAPIKeyRequest{
		KeyID:  "key_123",
		TeamID: "team_123",
		Name:   "renamed-key",
	})
	if err != nil {
		t.Fatalf("UpdateAIGatewayAPIKey() error = %v", err)
	}
	if string(got["name"]) != `"renamed-key"` || len(got) != 1 {
		t.Fatalf("update request = %v", got)
	}
	if key.Name != "renamed-key" {
		t.Fatalf("name = %q, want renamed-key", key.Name)
	}
}

func TestUpdateAIGatewayAPIKeyQuota(t *testing.T) {
	var got map[string]json.RawMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/v1/api-keys/key_123/quota" {
			t.Fatalf("request = %s %s, want PATCH /v1/api-keys/key_123/quota", r.Method, r.URL.Path)
		}
		if teamID := r.URL.Query().Get("teamId"); teamID != "team_123" {
			t.Fatalf("teamId = %q, want team_123", teamID)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		_, _ = w.Write([]byte(`{
			"apiKey": {"id": "key_123"},
			"quota": {"limitAmount": 500, "refreshPeriod": "monthly", "alertThresholds": []}
		}`))
	}))
	t.Cleanup(server.Close)

	limitAmount := float64(500)
	thresholds := []int64{}
	quota, err := New("TOKEN").WithBaseURL(server.URL).UpdateAIGatewayAPIKeyQuota(context.Background(), UpdateAIGatewayAPIKeyQuotaRequest{
		KeyID:           "key_123",
		TeamID:          "team_123",
		LimitAmount:     &limitAmount,
		RefreshPeriod:   "monthly",
		AlertThresholds: &thresholds,
	})
	if err != nil {
		t.Fatalf("UpdateAIGatewayAPIKeyQuota() error = %v", err)
	}
	if string(got["limitAmount"]) != `500` || string(got["refreshPeriod"]) != `"monthly"` || string(got["alertThresholds"]) != `[]` {
		t.Fatalf("quota update request = %v", got)
	}
	if _, ok := got["archived"]; ok {
		t.Fatalf("quota update request unexpectedly contains archived: %s", got["archived"])
	}
	if quota == nil || quota.LimitAmount != 500 || quota.RefreshPeriod != "monthly" {
		t.Fatalf("quota = %#v", quota)
	}
}

func TestArchiveAIGatewayAPIKeyQuota(t *testing.T) {
	var got map[string]json.RawMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/v1/api-keys/key_123/quota" {
			t.Fatalf("request = %s %s, want PATCH /v1/api-keys/key_123/quota", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		_, _ = w.Write([]byte(`{
			"apiKey": {"id": "key_123"},
			"quota": {"limitAmount": 500, "refreshPeriod": "monthly"}
		}`))
	}))
	t.Cleanup(server.Close)

	archived := true
	_, err := New("TOKEN").WithBaseURL(server.URL).UpdateAIGatewayAPIKeyQuota(context.Background(), UpdateAIGatewayAPIKeyQuotaRequest{
		KeyID:    "key_123",
		TeamID:   "team_123",
		Archived: &archived,
	})
	if err != nil {
		t.Fatalf("UpdateAIGatewayAPIKeyQuota() error = %v", err)
	}
	if string(got["archived"]) != `true` || len(got) != 1 {
		t.Fatalf("archive request = %v", got)
	}
}

func TestDeleteAIGatewayAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/v1/api-keys/key_123" {
			t.Fatalf("request = %s %s, want DELETE /v1/api-keys/key_123", r.Method, r.URL.Path)
		}
		if teamID := r.URL.Query().Get("teamId"); teamID != "team_123" {
			t.Fatalf("teamId = %q, want team_123", teamID)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	if err := New("TOKEN").WithBaseURL(server.URL).DeleteAIGatewayAPIKey(context.Background(), "key_123", "team_123"); err != nil {
		t.Fatalf("DeleteAIGatewayAPIKey() error = %v", err)
	}
}

func assertAIGatewayAPIKeyResponse(t *testing.T, key AIGatewayAPIKey) {
	t.Helper()
	if key.ID != "key_123" || key.Name != "workflow-github-actions" || key.PartialKey != "abc" {
		t.Fatalf("API key = %#v", key)
	}
	if key.TeamID != "team_123" {
		t.Fatalf("API key metadata = %#v", key)
	}
}
