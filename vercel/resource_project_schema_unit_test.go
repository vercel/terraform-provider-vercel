package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v5/client"
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

func TestProjectBuildMachineTypeAcceptsBasic(t *testing.T) {
	res := newProjectResource()
	resp := &resource.SchemaResponse{}
	res.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attribute := resp.Schema.Attributes["build_machine_type"].(resourceschema.StringAttribute)
	for _, v := range attribute.Validators {
		validationResp := &validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{
			Path:        path.Root("build_machine_type"),
			ConfigValue: types.StringValue("basic"),
		}, validationResp)
		if validationResp.Diagnostics.HasError() {
			t.Fatalf("basic should be valid: %s", validationResp.Diagnostics)
		}
	}
}

func TestDataSourceProtectionBypassForAutomationReturnsMultipleBypasses(t *testing.T) {
	enabled, secrets := dataSourceProtectionBypassForAutomation(client.ProjectResponse{
		ProtectionBypass: map[string]client.ProtectionBypass{
			"abcdefghijklmnopqrstuvwxyz123456": {Scope: "automation-bypass"},
			"12345678901234567890123456789012": {Scope: "automation-bypass"},
		},
	})
	if enabled.IsNull() || !enabled.ValueBool() {
		t.Fatalf("enabled = %v, want true", enabled)
	}
	want := types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("12345678901234567890123456789012"),
		types.StringValue("abcdefghijklmnopqrstuvwxyz123456"),
	})
	if !secrets.Equal(want) {
		t.Fatalf("secrets = %v, want %v", secrets, want)
	}
}

func TestDataSourceProtectionBypassForAutomationReturnsSingleBypass(t *testing.T) {
	enabled, secrets := dataSourceProtectionBypassForAutomation(client.ProjectResponse{
		ProtectionBypass: map[string]client.ProtectionBypass{
			"abcdefghijklmnopqrstuvwxyz123456": {Scope: "automation-bypass"},
		},
	})
	if enabled.IsNull() || !enabled.ValueBool() {
		t.Fatalf("enabled = %v, want true", enabled)
	}
	want := types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("abcdefghijklmnopqrstuvwxyz123456"),
	})
	if !secrets.Equal(want) {
		t.Fatalf("secrets = %v, want %v", secrets, want)
	}
}
