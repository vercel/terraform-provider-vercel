package vercel

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

func TestProjectResourceSchemaOmitsDeprecatedAutomationBypassAttributes(t *testing.T) {
	res := newProjectResource()

	resp := &resource.SchemaResponse{}
	res.Schema(context.Background(), resource.SchemaRequest{}, resp)

	if _, ok := resp.Schema.Attributes["protection_bypass_for_automation"]; ok {
		t.Fatal("protection_bypass_for_automation should not be present in vercel_project schema")
	}
	if _, ok := resp.Schema.Attributes["protection_bypass_for_automation_secret"]; ok {
		t.Fatal("protection_bypass_for_automation_secret should not be present in vercel_project schema")
	}
}

func TestDataSourceProtectionBypassForAutomationRejectsMultipleBypasses(t *testing.T) {
	_, _, err := dataSourceProtectionBypassForAutomation(client.ProjectResponse{
		ProtectionBypass: map[string]client.ProtectionBypass{
			"abcdefghijklmnopqrstuvwxyz123456": {Scope: "automation-bypass"},
			"12345678901234567890123456789012": {Scope: "automation-bypass"},
		},
	})
	if err == nil {
		t.Fatal("dataSourceProtectionBypassForAutomation() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "multiple protection bypasses") {
		t.Fatalf("dataSourceProtectionBypassForAutomation() error = %q, want multiple protection bypasses", err)
	}
}

func TestDataSourceProtectionBypassForAutomationReturnsSingleBypass(t *testing.T) {
	enabled, secret, err := dataSourceProtectionBypassForAutomation(client.ProjectResponse{
		ProtectionBypass: map[string]client.ProtectionBypass{
			"abcdefghijklmnopqrstuvwxyz123456": {Scope: "automation-bypass"},
		},
	})
	if err != nil {
		t.Fatalf("dataSourceProtectionBypassForAutomation() error = %v", err)
	}
	if enabled.IsNull() || !enabled.ValueBool() {
		t.Fatalf("enabled = %v, want true", enabled)
	}
	if secret != types.StringValue("abcdefghijklmnopqrstuvwxyz123456") {
		t.Fatalf("secret = %v, want %v", secret, types.StringValue("abcdefghijklmnopqrstuvwxyz123456"))
	}
}
