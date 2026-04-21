package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

func TestConvertResponseToProjectEnvironmentVariableKeepsWriteOnlyValueNull(t *testing.T) {
	t.Parallel()

	result := convertResponseToProjectEnvironmentVariable(
		client.EnvironmentVariable{
			ID:      "env_123",
			Key:     "SECRET",
			Value:   "bar-wo",
			Target:  []string{"production"},
			Type:    "sensitive",
			Comment: "test comment",
		},
		types.StringValue("prj_123"),
		types.StringNull(),
	)

	if !result.Value.IsNull() {
		t.Fatalf("Value = %v, want null", result.Value)
	}

	if !result.ValueWO.IsNull() {
		t.Fatalf("ValueWO = %v, want null", result.ValueWO)
	}
}

func TestConvertResponseToProjectEnvironmentVariableUsesProvidedSensitiveValueWhenAvailable(t *testing.T) {
	t.Parallel()

	result := convertResponseToProjectEnvironmentVariable(
		client.EnvironmentVariable{
			ID:      "env_123",
			Key:     "SECRET",
			Value:   "bar-new",
			Target:  []string{"production"},
			Type:    "sensitive",
			Comment: "test comment",
		},
		types.StringValue("prj_123"),
		types.StringValue("bar-new"),
	)

	if result.Value.IsNull() {
		t.Fatal("Value is null, want provided value")
	}

	if got := result.Value.ValueString(); got != "bar-new" {
		t.Fatalf("Value = %q, want %q", got, "bar-new")
	}
}

func TestProjectEnvironmentVariableResourceSchemaRequiresSensitive(t *testing.T) {
	res := newProjectEnvironmentVariableResource()

	resp := &resource.SchemaResponse{}
	res.Schema(context.Background(), resource.SchemaRequest{}, resp)

	sensitiveAttr, ok := resp.Schema.Attributes["sensitive"].(schema.BoolAttribute)
	if !ok {
		t.Fatalf("sensitive attribute has unexpected type: %T", resp.Schema.Attributes["sensitive"])
	}

	assertBoolRequired(t, sensitiveAttr, "sensitive")
}

func TestProjectEnvironmentVariableSensitiveSemantics(t *testing.T) {
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
			env := ProjectEnvironmentVariable{Sensitive: tt.sensitive}

			if got := env.isExplicitlyNonSensitive(); got != tt.wantExplicitlyNonSensitive {
				t.Fatalf("isExplicitlyNonSensitive() = %t, want %t", got, tt.wantExplicitlyNonSensitive)
			}

			if got := env.isSensitive(); got != tt.wantSensitive {
				t.Fatalf("isSensitive() = %t, want %t", got, tt.wantSensitive)
			}
		})
	}
}

func TestProjectEnvironmentVariableHasTarget(t *testing.T) {
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
			env := ProjectEnvironmentVariable{Target: tt.target}

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

func TestProjectEnvironmentVariableModifyPlanSkipsPolicyValidationForExistingResource(t *testing.T) {
	ctx := context.Background()
	policy := "on"
	res := &projectEnvironmentVariableResource{
		client: client.New("").WithTeam(client.Team{
			SensitiveEnvironmentVariablePolicy: &policy,
		}),
	}

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	config := ProjectEnvironmentVariable{
		Target:               stringSet("production"),
		CustomEnvironmentIDs: types.SetNull(types.StringType),
		GitBranch:            types.StringNull(),
		Key:                  types.StringValue("EXAMPLE"),
		Value:                types.StringValue("value"),
		ValueWO:              types.StringNull(),
		TeamID:               types.StringNull(),
		ProjectID:            types.StringValue("prj_123"),
		ID:                   types.StringNull(),
		Sensitive:            types.BoolValue(false),
		Comment:              types.StringNull(),
	}

	plan := config
	plan.ID = types.StringValue("env_123")

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

func TestProjectEnvironmentVariableModifyPlanUsesPlannedDevelopmentTarget(t *testing.T) {
	ctx := context.Background()
	res := &projectEnvironmentVariableResource{}

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	config := ProjectEnvironmentVariable{
		Target:               types.SetNull(types.StringType),
		CustomEnvironmentIDs: types.SetNull(types.StringType),
		GitBranch:            types.StringNull(),
		Key:                  types.StringValue("EXAMPLE"),
		Value:                types.StringValue("value"),
		ValueWO:              types.StringNull(),
		TeamID:               types.StringNull(),
		ProjectID:            types.StringValue("prj_123"),
		ID:                   types.StringNull(),
		Sensitive:            types.BoolValue(true),
		Comment:              types.StringNull(),
	}

	plan := config
	plan.Target = stringSet("development")
	plan.ID = types.StringValue("env_123")
	plan.Sensitive = types.BoolValue(true)

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

	if !resp.Diagnostics.HasError() {
		t.Fatal("ModifyPlan() expected diagnostics, got none")
	}

	if len(resp.Diagnostics) != 1 {
		t.Fatalf("ModifyPlan() returned %d diagnostics, want 1", len(resp.Diagnostics))
	}

	if got := resp.Diagnostics[0].Detail(); got != "Environment variables targeting `development` must explicitly set `sensitive = false`." {
		t.Fatalf("ModifyPlan() diagnostic detail = %q, want %q", got, "Environment variables targeting `development` must explicitly set `sensitive = false`.")
	}
}

func stringSet(values ...string) types.Set {
	targets := make([]attr.Value, 0, len(values))
	for _, value := range values {
		targets = append(targets, types.StringValue(value))
	}

	return types.SetValueMust(types.StringType, targets)
}
