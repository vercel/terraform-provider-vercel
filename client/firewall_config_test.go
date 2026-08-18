package client_test

import (
	"context"
	"net/http"
	"testing"

	vercelclient "github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestGetFirewallConfigPreservesIdentity(t *testing.T) {
	t.Parallel()

	client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/security/firewall/config/active", "team_123", nil)
		if value := r.URL.Query().Get("projectId"); value != "prj_123" {
			t.Fatalf("expected projectId %q, got %q", "prj_123", value)
		}
		_, _ = w.Write([]byte(`{"firewallEnabled":true}`))
	})

	config, err := client.GetFirewallConfig(context.Background(), "prj_123", "team_123")
	if err != nil {
		t.Fatalf("GetFirewallConfig returned error: %v", err)
	}
	if config.ProjectID != "prj_123" {
		t.Fatalf("expected project ID %q, got %q", "prj_123", config.ProjectID)
	}
	if config.TeamID != "team_123" {
		t.Fatalf("expected team ID %q, got %q", "team_123", config.TeamID)
	}
}

func TestPutFirewallConfigPreservesIdentity(t *testing.T) {
	t.Parallel()

	client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "PUT", "/v1/security/firewall/config", "team_123", map[string]any{
			"firewallEnabled": true,
		})
		if value := r.URL.Query().Get("projectId"); value != "prj_123" {
			t.Fatalf("expected projectId %q, got %q", "prj_123", value)
		}
		_, _ = w.Write([]byte(`{"active":{"firewallEnabled":true}}`))
	})

	config, err := client.PutFirewallConfig(context.Background(), vercelclient.FirewallConfig{
		ProjectID: "prj_123",
		TeamID:    "team_123",
		Enabled:   true,
	})
	if err != nil {
		t.Fatalf("PutFirewallConfig returned error: %v", err)
	}
	if config.ProjectID != "prj_123" {
		t.Fatalf("expected project ID %q, got %q", "prj_123", config.ProjectID)
	}
	if config.TeamID != "team_123" {
		t.Fatalf("expected team ID %q, got %q", "team_123", config.TeamID)
	}
}

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
