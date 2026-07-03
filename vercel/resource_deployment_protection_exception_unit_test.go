package vercel

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v5/client"
)

func TestResponseToDeploymentProtectionException(t *testing.T) {
	response := client.AliasResponse{
		Alias:     "preview.example.vercel.app",
		ProjectID: "prj_123",
		TeamID:    "team_123",
		ProtectionBypass: map[string]client.ProtectionBypass{
			"*": {
				Scope:     "alias-protection-override",
				CreatedAt: 123,
				CreatedBy: "user_123",
			},
		},
	}

	result, ok := responseToDeploymentProtectionException(response, DeploymentProtectionException{
		ProjectID: types.StringValue("prj_123"),
		TeamID:    types.StringValue("team_state"),
		Alias:     types.StringValue("state.example.vercel.app"),
	})

	if !ok {
		t.Fatal("responseToDeploymentProtectionException returned ok=false")
	}
	if result.ID.ValueString() != "prj_123/preview.example.vercel.app" {
		t.Fatalf("ID = %q, want prj_123/preview.example.vercel.app", result.ID.ValueString())
	}
	if result.ProjectID.ValueString() != "prj_123" {
		t.Fatalf("ProjectID = %q, want prj_123", result.ProjectID.ValueString())
	}
	if result.TeamID.ValueString() != "team_123" {
		t.Fatalf("TeamID = %q, want team_123", result.TeamID.ValueString())
	}
	if result.Alias.ValueString() != "preview.example.vercel.app" {
		t.Fatalf("Alias = %q, want preview.example.vercel.app", result.Alias.ValueString())
	}
	if result.CreatedAt.ValueInt64() != 123 {
		t.Fatalf("CreatedAt = %d, want 123", result.CreatedAt.ValueInt64())
	}
	if result.CreatedBy.ValueString() != "user_123" {
		t.Fatalf("CreatedBy = %q, want user_123", result.CreatedBy.ValueString())
	}
	if result.Scope.ValueString() != "alias-protection-override" {
		t.Fatalf("Scope = %q, want alias-protection-override", result.Scope.ValueString())
	}
}

func TestResponseToDeploymentProtectionExceptionRequiresOverrideScope(t *testing.T) {
	_, ok := responseToDeploymentProtectionException(client.AliasResponse{
		ProtectionBypass: map[string]client.ProtectionBypass{
			"*": {
				Scope: "shareable-link",
			},
		},
	}, DeploymentProtectionException{})

	if ok {
		t.Fatal("responseToDeploymentProtectionException returned ok=true for non-override scope")
	}
}
