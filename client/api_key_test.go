package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const apiKeyResponse = `{
	"apiKey": {
		"id": "key_123",
		"name": "workflow-github-actions",
		"partialKey": "abc",
		"teamId": "team_123",
		"purpose": "ai-gateway",
		"createdAt": 1700000000000
	}
}`

func TestCreateAPIKey(t *testing.T) {
	var got CreateAPIKeyRequest
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

	key, err := New("TOKEN").WithBaseURL(server.URL).CreateAPIKey(context.Background(), CreateAPIKeyRequest{
		Name:    "workflow-github-actions",
		Purpose: "ai-gateway",
		TeamID:  "team_123",
	})
	if err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}
	if got.TeamID != "" || got.Name != "workflow-github-actions" || got.Purpose != "ai-gateway" {
		t.Fatalf("create request = %#v", got)
	}
	assertAPIKeyResponse(t, key)
	if key.APIKeyString == nil || *key.APIKeyString != "vck_secret" {
		t.Fatalf("APIKeyString = %#v, want vck_secret", key.APIKeyString)
	}
}

func TestGetAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/api-keys/key_123" {
			t.Fatalf("request = %s %s, want GET /v1/api-keys/key_123", r.Method, r.URL.Path)
		}
		if teamID := r.URL.Query().Get("teamId"); teamID != "team_123" {
			t.Fatalf("teamId = %q, want team_123", teamID)
		}
		_, _ = w.Write([]byte(apiKeyResponse))
	}))
	t.Cleanup(server.Close)

	key, err := New("TOKEN").WithBaseURL(server.URL).GetAPIKey(context.Background(), "key_123", "team_123")
	if err != nil {
		t.Fatalf("GetAPIKey() error = %v", err)
	}
	assertAPIKeyResponse(t, key)
	if key.APIKeyString != nil {
		t.Fatalf("APIKeyString = %#v, want nil", key.APIKeyString)
	}
}

func TestDeleteAPIKey(t *testing.T) {
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

	if err := New("TOKEN").WithBaseURL(server.URL).DeleteAPIKey(context.Background(), "key_123", "team_123"); err != nil {
		t.Fatalf("DeleteAPIKey() error = %v", err)
	}
}

func assertAPIKeyResponse(t *testing.T, key APIKey) {
	t.Helper()
	if key.ID != "key_123" || key.Name != "workflow-github-actions" || key.PartialKey != "abc" {
		t.Fatalf("API key = %#v", key)
	}
	if key.TeamID != "team_123" || key.Purpose != "ai-gateway" || key.CreatedAt != 1700000000000 {
		t.Fatalf("API key metadata = %#v", key)
	}
}
