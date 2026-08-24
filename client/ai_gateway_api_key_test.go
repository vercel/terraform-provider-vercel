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
		"createdAt": 1700000000000
	}
}`

func TestCreateAIGatewayAPIKey(t *testing.T) {
	var got CreateAIGatewayAPIKeyRequest
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
				"createdAt": 1700000000000
			}
		}`))
	}))
	t.Cleanup(server.Close)

	key, err := New("TOKEN").WithBaseURL(server.URL).CreateAIGatewayAPIKey(context.Background(), CreateAIGatewayAPIKeyRequest{
		Name:    "workflow-github-actions",
		Purpose: "ai-gateway",
		TeamID:  "team_123",
	})
	if err != nil {
		t.Fatalf("CreateAIGatewayAPIKey() error = %v", err)
	}
	if got.TeamID != "" || got.Name != "workflow-github-actions" || got.Purpose != "ai-gateway" {
		t.Fatalf("create request = %#v", got)
	}
	assertAIGatewayAPIKeyResponse(t, key)
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
	if key.APIKeyString != nil {
		t.Fatalf("APIKeyString = %#v, want nil", key.APIKeyString)
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
