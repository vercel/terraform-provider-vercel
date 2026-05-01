package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetMicrofrontendGroupMembershipReturnsNotFoundWhenProjectMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		fmt.Fprintln(w, `{"groups":[{"group":{"id":"group_123","name":"Group","slug":"group","team_id":"team_123","projects":{}},"projects":[{"id":"prj_other","microfrontends":{"groupIds":["group_123"],"enabled":true,"isDefaultApp":false,"routeObservabilityToThisProject":false}}]}]}`)
	}))
	t.Cleanup(server.Close)

	cl := New("INVALID")
	cl.baseURL = server.URL
	_, err := cl.GetMicrofrontendGroupMembership(context.Background(), "team_123", "group_123", "prj_123")
	if !NotFound(err) {
		t.Fatalf("GetMicrofrontendGroupMembership() error = %v, want NotFound", err)
	}
}

func TestPatchMicrofrontendGroupMembershipSendsFalseRouteObservability(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("method = %s, want PATCH", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		fmt.Fprintln(w, `{"microfrontends":{"enabled":true,"isDefaultApp":false,"routeObservabilityToThisProject":false}}`)
	}))
	t.Cleanup(server.Close)

	cl := New("INVALID")
	cl.baseURL = server.URL
	_, err := cl.PatchMicrofrontendGroupMembership(context.Background(), MicrofrontendGroupMembership{
		ProjectID:                       "prj_123",
		MicrofrontendGroupID:            "mfe_123",
		Enabled:                         true,
		RouteObservabilityToThisProject: false,
	})
	if err != nil {
		t.Fatalf("PatchMicrofrontendGroupMembership() error = %v", err)
	}

	got, ok := payload["routeObservabilityToThisProject"].(bool)
	if !ok {
		t.Fatalf("routeObservabilityToThisProject was not included as a bool: %#v", payload)
	}
	if got {
		t.Fatalf("routeObservabilityToThisProject = true, want false")
	}
}
