package client_test

import (
	"context"
	"net/http"
	"testing"

	vercelclient "github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestUpdateDeploymentProtectionExceptionCreate(t *testing.T) {
	t.Parallel()

	client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "PATCH", "/v3/aliases/preview.example.vercel.app/protection-bypass", "team_123", map[string]any{
			"override": map[string]any{
				"scope":  "alias-protection-override",
				"action": "create",
			},
		})
		_, _ = w.Write([]byte(`{"protectionBypass":{"*":{"createdAt":123,"createdBy":"user_123","scope":"alias-protection-override"}}}`))
	})

	bypasses, err := client.UpdateDeploymentProtectionException(context.Background(), vercelclient.UpdateDeploymentProtectionExceptionRequest{
		Alias:  "preview.example.vercel.app",
		TeamID: "team_123",
		Action: "create",
	})
	if err != nil {
		t.Fatalf("UpdateDeploymentProtectionException returned error: %v", err)
	}

	override := bypasses["*"]
	if override.Scope != "alias-protection-override" {
		t.Fatalf("override scope = %q, want alias-protection-override", override.Scope)
	}
	if override.CreatedAt != 123 {
		t.Fatalf("override createdAt = %d, want 123", override.CreatedAt)
	}
	if override.CreatedBy != "user_123" {
		t.Fatalf("override createdBy = %q, want user_123", override.CreatedBy)
	}
}

func TestUpdateDeploymentProtectionExceptionRevokeUsesConfiguredTeam(t *testing.T) {
	t.Parallel()

	client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "PATCH", "/v3/aliases/preview.example.vercel.app/protection-bypass", "team_default", map[string]any{
			"override": map[string]any{
				"scope":  "alias-protection-override",
				"action": "revoke",
			},
		})
		_, _ = w.Write([]byte(`{"protectionBypass":{}}`))
	}).WithTeam(vercelclient.Team{ID: "team_default"})

	_, err := client.UpdateDeploymentProtectionException(context.Background(), vercelclient.UpdateDeploymentProtectionExceptionRequest{
		Alias:  "preview.example.vercel.app",
		Action: "revoke",
	})
	if err != nil {
		t.Fatalf("UpdateDeploymentProtectionException returned error: %v", err)
	}
}

func TestGetAliasReadsDeploymentProtectionException(t *testing.T) {
	t.Parallel()

	client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v4/aliases/preview.example.vercel.app", "team_123", nil)
		_, _ = w.Write([]byte(`{
			"uid":"alias_123",
			"alias":"preview.example.vercel.app",
			"deploymentId":"dpl_123",
			"projectId":"prj_123",
			"protectionBypass":{"*":{"createdAt":123,"createdBy":"user_123","scope":"alias-protection-override"}}
		}`))
	})

	alias, err := client.GetAlias(context.Background(), "preview.example.vercel.app", "team_123")
	if err != nil {
		t.Fatalf("GetAlias returned error: %v", err)
	}

	if alias.ProjectID != "prj_123" {
		t.Fatalf("alias ProjectID = %q, want prj_123", alias.ProjectID)
	}
	override := alias.ProtectionBypass["*"]
	if override.Scope != "alias-protection-override" {
		t.Fatalf("override scope = %q, want alias-protection-override", override.Scope)
	}
}
