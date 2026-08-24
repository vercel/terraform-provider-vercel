package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetProjectTracing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/drains/tracing/config" {
			t.Fatalf("request = %s %s, want GET /v1/drains/tracing/config", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("projectId"); got != "prj_123" {
			t.Fatalf("projectId = %q, want prj_123", got)
		}
		if got := r.URL.Query().Get("teamId"); got != "team_123" {
			t.Fatalf("teamId = %q, want team_123", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"enabled":true,"sampling":[{"type":"head_sampling","rate":0.25,"env":"production","requestPath":"/api"}]}`))
	}))
	t.Cleanup(server.Close)

	tracing, err := New("TOKEN").WithBaseURL(server.URL).GetProjectTracing(context.Background(), "prj_123", "team_123")
	if err != nil {
		t.Fatalf("GetProjectTracing() error = %v", err)
	}
	if !tracing.Enabled || tracing.TeamID != "team_123" || tracing.ProjectID != "prj_123" {
		t.Fatalf("tracing = %#v", tracing)
	}
	if len(tracing.SamplingRules) != 1 || tracing.SamplingRules[0].Rate != 0.25 || tracing.SamplingRules[0].Environment != "production" || tracing.SamplingRules[0].RequestPath != "/api" {
		t.Fatalf("sampling rules = %#v", tracing.SamplingRules)
	}
}

func TestUpdateProjectTracing(t *testing.T) {
	var payload updateProjectTracingPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/v1/drains/tracing/config" {
			t.Fatalf("request = %s %s, want PUT /v1/drains/tracing/config", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"enabled":true,"sampling":[{"type":"head_sampling","rate":0.5,"env":"preview"}]}`))
	}))
	t.Cleanup(server.Close)

	tracing, err := New("TOKEN").WithBaseURL(server.URL).UpdateProjectTracing(context.Background(), ProjectTracing{
		TeamID:    "team_123",
		ProjectID: "prj_123",
		Enabled:   true,
		SamplingRules: []TraceDrainSamplingRule{
			{Rate: 0.5, Environment: "preview"},
		},
	})
	if err != nil {
		t.Fatalf("UpdateProjectTracing() error = %v", err)
	}
	if !payload.Enabled || len(payload.Sampling) != 1 || payload.Sampling[0].Type != "head_sampling" || payload.Sampling[0].Rate != 0.5 || payload.Sampling[0].Env != "preview" {
		t.Fatalf("payload = %#v", payload)
	}
	if !tracing.Enabled || len(tracing.SamplingRules) != 1 || tracing.SamplingRules[0].Rate != 0.5 {
		t.Fatalf("tracing = %#v", tracing)
	}
}

func TestUpdateProjectTracingClearsSamplingRules(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		if got := string(payload["sampling"]); got != "[]" {
			t.Fatalf("sampling = %s, want []", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"enabled":true,"sampling":[]}`))
	}))
	t.Cleanup(server.Close)

	_, err := New("TOKEN").WithBaseURL(server.URL).UpdateProjectTracing(context.Background(), ProjectTracing{
		ProjectID: "prj_123",
		Enabled:   true,
	})
	if err != nil {
		t.Fatalf("UpdateProjectTracing() error = %v", err)
	}
}

func TestUpdateProjectTracingDisablesTracing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload updateProjectTracingPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		if payload.Enabled {
			t.Fatal("enabled = true, want false")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"enabled":false,"sampling":[]}`))
	}))
	t.Cleanup(server.Close)

	_, err := New("TOKEN").WithBaseURL(server.URL).UpdateProjectTracing(context.Background(), ProjectTracing{
		ProjectID: "prj_123",
		Enabled:   false,
	})
	if err != nil {
		t.Fatalf("UpdateProjectTracing() error = %v", err)
	}
}
