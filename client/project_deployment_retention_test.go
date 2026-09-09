package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetDeploymentRetentionTeamOwnership(t *testing.T) {
	for _, tc := range []struct {
		name, response, wantTeam string
		wantPreview              int
	}{
		{"team with retention", `{"accountId":"team_owner","deploymentExpiration":{"expirationDays":30}}`, "team_owner", 30},
		{"team with defaults", `{"accountId":"team_owner"}`, "team_owner", 36500},
		{"personal with defaults", `{"accountId":"user_owner"}`, "", 36500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/v2/projects/prj_123" || r.URL.RawQuery != "" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.response))
			}))
			t.Cleanup(server.Close)
			out, err := New("TOKEN").WithBaseURL(server.URL).GetDeploymentRetention(context.Background(), "prj_123", "")
			if err != nil {
				t.Fatal(err)
			}
			if out.TeamID != tc.wantTeam || out.ExpirationPreview != tc.wantPreview {
				t.Fatalf("unexpected retention: %#v; want team %q, preview %d", out, tc.wantTeam, tc.wantPreview)
			}
		})
	}
}

func TestGetDeploymentRetentionReturnsReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"Project not found"}}`))
	}))
	t.Cleanup(server.Close)
	_, err := New("TOKEN").WithBaseURL(server.URL).GetDeploymentRetention(context.Background(), "prj_missing", "")
	if !NotFound(err) {
		t.Fatalf("error = %v, want not found", err)
	}
}
