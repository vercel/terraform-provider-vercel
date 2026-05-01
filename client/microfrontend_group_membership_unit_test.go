package client

import (
	"context"
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
