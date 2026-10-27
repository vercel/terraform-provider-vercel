package client_test

import (
	"context"
	"fmt"
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

func TestFirewallConfigAiBotsCategories(t *testing.T) {
	t.Parallel()
	groups := map[string]vercelclient.CoreRuleSet{
		"ai_crawlers":   {Active: true, Action: "deny"},
		"ai_assistants": {Active: false, Action: "log"},
	}
	c := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			assertRequest(t, r, "PUT", "/v1/security/firewall/config", "team_123", map[string]any{
				"firewallEnabled": true,
				"managedRules": map[string]any{"ai_bots": map[string]any{
					"active":     true,
					"ruleGroups": groups,
				}},
			})
			fmt.Fprint(w, `{"active":{"firewallEnabled":true,"managedRules":{"ai_bots":{"active":true,"ruleGroups":{"ai_crawlers":{"active":true,"action":"deny"},"ai_assistants":{"active":false,"action":"log"}}}}}}`)
			return
		}
		assertRequest(t, r, "GET", "/v1/security/firewall/config/active", "team_123", nil)
		fmt.Fprint(w, `{"firewallEnabled":true,"managedRules":{"ai_bots":{"active":true,"ruleGroups":{"ai_crawlers":{"active":true,"action":"deny"},"ai_assistants":{"active":false,"action":"log"}}}}}`)
	})
	cfg := vercelclient.FirewallConfig{
		ProjectID: "prj_123", TeamID: "team_123", Enabled: true,
		ManagedRulesets: map[string]vercelclient.ManagedRule{"ai_bots": {Active: true, RuleGroups: groups}},
	}
	created, err := c.PutFirewallConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	read, err := c.GetFirewallConfig(context.Background(), cfg.ProjectID, cfg.TeamID)
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range []vercelclient.FirewallConfig{created, read} {
		rule := result.ManagedRulesets["ai_bots"]
		if rule.Action != "" || !rule.Active || len(rule.RuleGroups) != 2 {
			t.Fatalf("unexpected AI Bots config: %+v", rule)
		}
		for id, want := range groups {
			if rule.RuleGroups[id] != want {
				t.Errorf("%s = %+v, want %+v", id, rule.RuleGroups[id], want)
			}
		}
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

func TestPutFirewallConfigBotID(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			calls := 0
			c := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				switch calls {
				case 1:
					body := map[string]any{"firewallEnabled": true}
					if enabled {
						body["botIdEnabled"] = true
					}
					assertRequest(t, r, "PUT", "/v1/security/firewall/config", "team_123", body)
					fmt.Fprint(w, `{"active":{"firewallEnabled":true}}`)
				case 2:
					assertRequest(t, r, "PATCH", "/v1/security/firewall/config", "team_123", map[string]any{"action": "botId.toggle", "id": nil, "value": true})
					fmt.Fprint(w, `{}`)
				case 3:
					assertRequest(t, r, "GET", "/v1/security/firewall/config/active", "team_123", nil)
					fmt.Fprint(w, `{"firewallEnabled":true,"botIdEnabled":true}`)
				default:
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
				}
				if r.URL.Query().Get("projectId") != "prj_123" {
					t.Error("missing project scope")
				}
			})
			got, err := c.PutFirewallConfig(context.Background(), vercelclient.FirewallConfig{ProjectID: "prj_123", TeamID: "team_123", Enabled: true, BotIDEnabled: enabled})
			if err != nil {
				t.Fatal(err)
			}
			wantCalls := 1
			if enabled {
				wantCalls = 3
			}
			if calls != wantCalls || got.BotIDEnabled != enabled {
				t.Fatalf("calls=%d, BotID=%v; want %d, %v", calls, got.BotIDEnabled, wantCalls, enabled)
			}
		})
	}
}

func TestPutFirewallConfigBotIDToggleFailure(t *testing.T) {
	calls := 0
	c := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			fmt.Fprint(w, `{"active":{"firewallEnabled":true}}`)
			return
		}
		if r.Method != "PATCH" {
			t.Fatalf("unexpected request: %s", r.Method)
		}
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"error":{"code":"forbidden","message":"BotID unavailable"}}`)
	})
	_, err := c.PutFirewallConfig(context.Background(), vercelclient.FirewallConfig{ProjectID: "prj_123", TeamID: "team_123", Enabled: true, BotIDEnabled: true})
	if err == nil {
		t.Fatal("expected BotID toggle error")
	}
}

func TestPutFirewallConfigDoesNotToggleWhenBotIDReturned(t *testing.T) {
	calls := 0
	c := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "PUT" {
			t.Fatalf("unexpected request: %s", r.Method)
		}
		fmt.Fprint(w, `{"active":{"firewallEnabled":true,"botIdEnabled":true}}`)
	})
	got, err := c.PutFirewallConfig(context.Background(), vercelclient.FirewallConfig{ProjectID: "prj_123", TeamID: "team_123", Enabled: true, BotIDEnabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || !got.BotIDEnabled {
		t.Fatalf("calls=%d, BotID=%v", calls, got.BotIDEnabled)
	}
}
