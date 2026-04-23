package client_test

import (
	"context"
	"net/http"
	"testing"

	vercelclient "github.com/vercel/terraform-provider-vercel/v4/client"
)

func TestUpdateFirewallConfig(t *testing.T) {
	t.Parallel()

	request := vercelclient.UpdateFirewallConfigRequest{
		ProjectID: "prj_123",
		TeamID:    "team_123",
		Action:    "rules.update",
		ID:        "rule_123",
		Value: map[string]any{
			"name":        "Updated Rule",
			"description": "Updated description",
			"active":      true,
		},
	}

	client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "PATCH", "/v1/security/firewall/config", "team_123", request)
		_, _ = w.Write([]byte(`{}`))
	})

	if err := client.UpdateFirewallConfig(context.Background(), request); err != nil {
		t.Fatalf("UpdateFirewallConfig returned error: %v", err)
	}
}

func TestUpdateFirewallConfigUsesConfiguredTeam(t *testing.T) {
	t.Parallel()

	client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "PATCH", "/v1/security/firewall/config", "team_default", map[string]any{
			"action": "rules.remove",
			"id":     "rule_123",
			"value":  nil,
		})
		_, _ = w.Write([]byte(`{}`))
	}).WithTeam(vercelclient.Team{ID: "team_default"})

	err := client.UpdateFirewallConfig(context.Background(), vercelclient.UpdateFirewallConfigRequest{
		ProjectID: "prj_123",
		Action:    "rules.remove",
		ID:        "rule_123",
		Value:     nil,
	})
	if err != nil {
		t.Fatalf("UpdateFirewallConfig returned error: %v", err)
	}
}
