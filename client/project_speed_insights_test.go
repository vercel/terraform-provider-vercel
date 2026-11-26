package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetProjectSpeedInsights(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     bool
	}{
		{name: "enabled", response: `{"id":"prj_123","speedInsights":{"id":"si_1","enabledAt":200}}`, want: true},
		{name: "re-enabled after disable", response: `{"id":"prj_123","speedInsights":{"id":"si_1","enabledAt":300,"disabledAt":200}}`, want: true},
		{name: "disabled", response: `{"id":"prj_123","speedInsights":{"id":"si_1","enabledAt":200,"disabledAt":300,"canceledAt":400}}`, want: false},
		{name: "never configured", response: `{"id":"prj_123"}`, want: false},
		{name: "legacy analytics enabled", response: `{"id":"prj_123","analytics":{"id":"an_1","enabledAt":200,"disabledAt":100}}`, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/v9/projects/prj_123" {
					t.Fatalf("request = %s %s, want GET /v9/projects/prj_123", r.Method, r.URL.Path)
				}
				if got := r.URL.Query().Get("teamId"); got != "team_123" {
					t.Fatalf("teamId = %q, want team_123", got)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.response))
			}))
			t.Cleanup(server.Close)

			out, err := New("TOKEN").WithBaseURL(server.URL).GetProjectSpeedInsights(context.Background(), "prj_123", "team_123")
			if err != nil {
				t.Fatalf("GetProjectSpeedInsights() error = %v", err)
			}
			if out.Enabled != tt.want || out.ProjectID != "prj_123" || out.TeamID != "team_123" {
				t.Fatalf("out = %#v, want Enabled=%v", out, tt.want)
			}
		})
	}
}

func TestUpdateProjectSpeedInsights(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		var payload toggleSpeedInsightsPayload
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/speed-insights/toggle" {
				t.Fatalf("request = %s %s, want POST /speed-insights/toggle", r.Method, r.URL.Path)
			}
			if got := r.URL.Query().Get("projectId"); got != "prj_123" {
				t.Fatalf("projectId = %q, want prj_123", got)
			}
			if got := r.URL.Query().Get("teamId"); got != "team_123" {
				t.Fatalf("teamId = %q, want team_123", got)
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(toggleSpeedInsightsResponse(payload))
		}))

		out, err := New("TOKEN").WithBaseURL(server.URL).UpdateProjectSpeedInsights(context.Background(), ProjectSpeedInsights{
			TeamID:    "team_123",
			ProjectID: "prj_123",
			Enabled:   enabled,
		})
		server.Close()
		if err != nil {
			t.Fatalf("UpdateProjectSpeedInsights(%v) error = %v", enabled, err)
		}
		if payload.Value != enabled {
			t.Fatalf("payload.Value = %v, want %v", payload.Value, enabled)
		}
		if out.Enabled != enabled || out.ProjectID != "prj_123" || out.TeamID != "team_123" {
			t.Fatalf("out = %#v", out)
		}
	}
}
