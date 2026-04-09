package vercel

import (
	"context"
	"testing"

	"github.com/vercel/terraform-provider-vercel/v4/client"
)

func TestAutomationBypassEnvVarSecret(t *testing.T) {
	t.Run("prefers explicit env var", func(t *testing.T) {
		manualNote := "manual"
		protectionBypass := map[string]client.ProtectionBypass{
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa": {
				Scope:    "automation-bypass",
				IsEnvVar: boolPointer(false),
				Note:     &manualNote,
			},
			"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb": {
				Scope: "automation-bypass",
			},
			"cccccccccccccccccccccccccccccccc": {
				Scope:    "automation-bypass",
				IsEnvVar: boolPointer(true),
			},
		}

		secret := automationBypassEnvVarSecret(protectionBypass)
		if secret != "cccccccccccccccccccccccccccccccc" {
			t.Fatalf("unexpected env var secret: %s", secret)
		}
	})

	t.Run("falls back to implicit env var", func(t *testing.T) {
		protectionBypass := map[string]client.ProtectionBypass{
			"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb": {
				Scope: "automation-bypass",
			},
		}

		secret := automationBypassEnvVarSecret(protectionBypass)
		if secret != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
			t.Fatalf("unexpected env var secret: %s", secret)
		}
	})
}

func TestProtectionBypassForAutomationSecretsSet(t *testing.T) {
	ctx := context.Background()
	note := "GitHub Actions"
	protectionBypass := map[string]client.ProtectionBypass{
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa": {
			Scope:    "automation-bypass",
			IsEnvVar: boolPointer(true),
			Note:     &note,
		},
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb": {
			Scope:    "automation-bypass",
			IsEnvVar: boolPointer(false),
		},
		"cccccccccccccccccccccccccccccccc": {
			Scope:           "integration-automation-bypass",
			IntegrationID:   "int_123",
			ConfigurationID: "cfg_123",
		},
	}

	set := protectionBypassForAutomationSecretsSet(automationBypassProtectionEntries(protectionBypass))

	var secrets []projectProtectionBypassForAutomationSecret
	diags := set.ElementsAs(ctx, &secrets, false)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if len(secrets) != 2 {
		t.Fatalf("unexpected secret count: %d", len(secrets))
	}

	desired := desiredProtectionBypassForAutomationSecretsMap(secrets)
	if _, ok := desired["cccccccccccccccccccccccccccccccc"]; ok {
		t.Fatal("expected integration automation bypass to be ignored")
	}
	if !desired["aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"].IsEnvVar {
		t.Fatal("expected selected bypass to be marked as env var")
	}
	if desired["aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"].Note == nil || *desired["aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"].Note != note {
		t.Fatalf("unexpected note: %#v", desired["aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"].Note)
	}
	if desired["bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"].IsEnvVar {
		t.Fatal("expected non-selected bypass to remain unset as the deployment env var")
	}
}
