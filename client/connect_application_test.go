package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetConnectApplication(t *testing.T) {
	for _, identifier := range []string{"scl_oidc", "oauth/company sso", "oauth/team?&#%"} {
		t.Run(identifier, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != "GET" || req.URL.Path != "/v1/connect/connectors/"+identifier {
					t.Errorf("request = %s %s", req.Method, req.URL.Path)
				}
				if strings.Contains(req.URL.EscapedPath(), "oauth/") {
					t.Error("UID slash must be URL-encoded")
				}
				if req.URL.Query().Get("teamId") != "team_1" {
					t.Error("missing team scope")
				}
				fmt.Fprint(w, `{"id":"scl_oidc","uid":"oauth/company sso","name":"Company SSO","type":"oauth","data":{"clientSecret":{"encrypted":true}},"tokens":{"secret":"not-state"}}`)
			}))
			defer server.Close()
			c := New("test").WithBaseURL(server.URL).WithTeam(Team{ID: "team_1"})
			application, err := c.GetConnectApplication(context.Background(), "", identifier)
			if err != nil {
				t.Fatal(err)
			}
			if application.ID != "scl_oidc" || application.UID != "oauth/company sso" || application.TeamID != "team_1" {
				t.Fatalf("application = %#v", application)
			}
		})
	}
}

func TestGetConnectApplicationNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
		fmt.Fprint(w, `{"error":{"code":"not_found","message":"Connector not found"}}`)
	}))
	defer server.Close()
	_, err := New("test").WithBaseURL(server.URL).GetConnectApplication(context.Background(), "team_1", "oauth/missing")
	var apiErr APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 404 {
		t.Fatalf("error = %v", err)
	}
}

func TestGetConnectApplicationRequiresTeam(t *testing.T) {
	_, err := New("test").GetConnectApplication(context.Background(), "", "scl_oidc")
	if err == nil || !strings.Contains(err.Error(), "team is required") {
		t.Fatalf("error = %v", err)
	}
}

func TestGetConnectApplicationErrorsDoNotExposeCredentials(t *testing.T) {
	for _, body := range []string{`{"id":123,"data":{"clientSecret":"private-secret"}}`, `{"error":{"code":"bad_request","message":"private-secret"}}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if strings.Contains(body, "bad_request") {
				w.WriteHeader(http.StatusBadRequest)
			}
			fmt.Fprint(w, body)
		}))
		_, err := New("test").WithBaseURL(server.URL).GetConnectApplication(context.Background(), "team_1", "scl_oidc")
		server.Close()
		if err == nil || strings.Contains(err.Error(), "private-secret") {
			t.Fatalf("unexpected diagnostic: %v", err)
		}
	}
}
