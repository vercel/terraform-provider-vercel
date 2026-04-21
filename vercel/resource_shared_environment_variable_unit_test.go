package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

func TestSharedEnvironmentVariableResourceSchemaRequiresSensitive(t *testing.T) {
	res := newSharedEnvironmentVariableResource()

	resp := &resource.SchemaResponse{}
	res.Schema(context.Background(), resource.SchemaRequest{}, resp)

	sensitiveAttr, ok := resp.Schema.Attributes["sensitive"].(schema.BoolAttribute)
	if !ok {
		t.Fatalf("sensitive attribute has unexpected type: %T", resp.Schema.Attributes["sensitive"])
	}

	assertBoolRequired(t, sensitiveAttr, "sensitive")
}

func TestSharedEnvironmentVariableSensitiveSemantics(t *testing.T) {
	tests := []struct {
		name                       string
		sensitive                  types.Bool
		wantExplicitlyNonSensitive bool
		wantSensitive              bool
	}{
		{
			name:                       "null is treated as sensitive",
			sensitive:                  types.BoolNull(),
			wantExplicitlyNonSensitive: false,
			wantSensitive:              true,
		},
		{
			name:                       "unknown is treated as sensitive",
			sensitive:                  types.BoolUnknown(),
			wantExplicitlyNonSensitive: false,
			wantSensitive:              true,
		},
		{
			name:                       "true stays sensitive",
			sensitive:                  types.BoolValue(true),
			wantExplicitlyNonSensitive: false,
			wantSensitive:              true,
		},
		{
			name:                       "false is explicitly non-sensitive",
			sensitive:                  types.BoolValue(false),
			wantExplicitlyNonSensitive: true,
			wantSensitive:              false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := SharedEnvironmentVariable{Sensitive: tt.sensitive}

			if got := env.isExplicitlyNonSensitive(); got != tt.wantExplicitlyNonSensitive {
				t.Fatalf("isExplicitlyNonSensitive() = %t, want %t", got, tt.wantExplicitlyNonSensitive)
			}

			if got := env.isSensitive(); got != tt.wantSensitive {
				t.Fatalf("isSensitive() = %t, want %t", got, tt.wantSensitive)
			}
		})
	}
}

func TestSharedEnvironmentVariableHasTarget(t *testing.T) {
	tests := []struct {
		name       string
		target     types.Set
		wantTarget bool
	}{
		{
			name:       "null target",
			target:     types.SetNull(types.StringType),
			wantTarget: false,
		},
		{
			name:       "unknown target",
			target:     types.SetUnknown(types.StringType),
			wantTarget: false,
		},
		{
			name:       "development target present",
			target:     stringSet("development", "preview"),
			wantTarget: true,
		},
		{
			name:       "development target absent",
			target:     stringSet("production", "preview"),
			wantTarget: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := SharedEnvironmentVariable{Target: tt.target}

			got, diags := env.hasTarget(context.Background(), "development")
			if diags.HasError() {
				t.Fatalf("hasTarget() returned diagnostics: %v", diags)
			}

			if got != tt.wantTarget {
				t.Fatalf("hasTarget() = %t, want %t", got, tt.wantTarget)
			}
		})
	}
}

func TestSharedEnvironmentVariableModifyPlanSkipsPolicyValidationForExistingResource(t *testing.T) {
	ctx := context.Background()
	policy := "on"
	res := &sharedEnvironmentVariableResource{
		client: client.New("").WithTeam(client.Team{
			SensitiveEnvironmentVariablePolicy: &policy,
		}),
	}

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	config := SharedEnvironmentVariable{
		Target:                       stringSet("production"),
		Key:                          types.StringValue("EXAMPLE"),
		Value:                        types.StringValue("value"),
		ValueWO:                      types.StringNull(),
		TeamID:                       types.StringNull(),
		ProjectIDs:                   stringSet("prj_123"),
		ID:                           types.StringNull(),
		Sensitive:                    types.BoolValue(false),
		Comment:                      types.StringNull(),
		ApplyToAllCustomEnvironments: types.BoolNull(),
	}

	plan := config
	plan.ID = types.StringValue("sev_123")

	configPlan := tfsdk.Plan{Schema: schemaResp.Schema}
	diags := configPlan.Set(ctx, config)
	if diags.HasError() {
		t.Fatalf("configPlan.Set() returned diagnostics: %v", diags)
	}

	plannedState := tfsdk.Plan{Schema: schemaResp.Schema}
	diags = plannedState.Set(ctx, plan)
	if diags.HasError() {
		t.Fatalf("plannedState.Set() returned diagnostics: %v", diags)
	}

	req := resource.ModifyPlanRequest{
		Config: tfsdk.Config{
			Raw:    configPlan.Raw,
			Schema: schemaResp.Schema,
		},
		Plan: plannedState,
	}
	resp := &resource.ModifyPlanResponse{
		Plan: plannedState,
	}

	res.ModifyPlan(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan() returned diagnostics: %v", resp.Diagnostics)
	}
}

func TestSharedEnvironmentVariableModifyPlanValidatesApplyAllCustomEnvironmentsAgainstPolicy(t *testing.T) {
	ctx := context.Background()
	policy := "on"
	res := &sharedEnvironmentVariableResource{
		client: client.New("").WithTeam(client.Team{
			SensitiveEnvironmentVariablePolicy: &policy,
		}),
	}

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	config := SharedEnvironmentVariable{
		Target:                       stringSet("development"),
		Key:                          types.StringValue("EXAMPLE"),
		Value:                        types.StringValue("value"),
		ValueWO:                      types.StringNull(),
		TeamID:                       types.StringNull(),
		ProjectIDs:                   stringSet("prj_123"),
		ID:                           types.StringNull(),
		Sensitive:                    types.BoolValue(false),
		Comment:                      types.StringNull(),
		ApplyToAllCustomEnvironments: types.BoolValue(true),
	}

	configPlan := tfsdk.Plan{Schema: schemaResp.Schema}
	diags := configPlan.Set(ctx, config)
	if diags.HasError() {
		t.Fatalf("configPlan.Set() returned diagnostics: %v", diags)
	}

	req := resource.ModifyPlanRequest{
		Config: tfsdk.Config{
			Raw:    configPlan.Raw,
			Schema: schemaResp.Schema,
		},
		Plan: configPlan,
	}
	resp := &resource.ModifyPlanResponse{
		Plan: configPlan,
	}

	res.ModifyPlan(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("ModifyPlan() expected diagnostics, got none")
	}

	if len(resp.Diagnostics) != 1 {
		t.Fatalf("ModifyPlan() returned %d diagnostics, want 1", len(resp.Diagnostics))
	}

	if got := resp.Diagnostics[0].Detail(); got != "This team has a policy that forces environment variables targeting `preview`, `production`, or custom environments to be sensitive. Set `sensitive = true` in your configuration." {
		t.Fatalf("ModifyPlan() diagnostic detail = %q", got)
	}
}
