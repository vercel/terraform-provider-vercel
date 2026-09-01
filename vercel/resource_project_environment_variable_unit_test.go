package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v5/client"
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
		types.Int64Value(2),
	)

	if !result.Value.IsNull() {
		t.Fatalf("Value = %v, want null", result.Value)
	}

	if !result.ValueWO.IsNull() {
		t.Fatalf("ValueWO = %v, want null", result.ValueWO)
	}

	if got := result.ValueWOVersion.ValueInt64(); got != 2 {
		t.Fatalf("ValueWOVersion = %d, want 2", got)
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
		types.Int64Null(),
	)

	if result.Value.IsNull() {
		t.Fatal("Value is null, want provided value")
	}

	if got := result.Value.ValueString(); got != "bar-new" {
		t.Fatalf("Value = %q, want %q", got, "bar-new")
	}
}

func TestProjectEnvironmentVariableUpdateRequestOmitsUnchangedWriteOnlyValue(t *testing.T) {
	t.Parallel()

	env := ProjectEnvironmentVariable{
		Target:               stringSet("production"),
		CustomEnvironmentIDs: types.SetNull(types.StringType),
		Value:                types.StringNull(),
		Sensitive:            types.BoolValue(true),
	}

	request, diags := env.toUpdateEnvironmentVariableRequest(context.Background(), types.StringNull())
	if diags.HasError() {
		t.Fatalf("toUpdateEnvironmentVariableRequest() returned diagnostics: %v", diags)
	}

	if request.Value != nil {
		t.Fatalf("Value = %q, want nil", *request.Value)
	}
}

func TestProjectEnvironmentVariableUpdateRequestIncludesChangedWriteOnlyValue(t *testing.T) {
	t.Parallel()

	env := ProjectEnvironmentVariable{
		Target:               stringSet("production"),
		CustomEnvironmentIDs: types.SetNull(types.StringType),
		Value:                types.StringNull(),
		Sensitive:            types.BoolValue(true),
	}

	request, diags := env.toUpdateEnvironmentVariableRequest(context.Background(), types.StringValue("new-secret"))
	if diags.HasError() {
		t.Fatalf("toUpdateEnvironmentVariableRequest() returned diagnostics: %v", diags)
	}

	if request.Value == nil || *request.Value != "new-secret" {
		t.Fatalf("Value = %v, want new-secret", request.Value)
	}
}

func TestShouldUpdateProjectEnvironmentVariableValueWO(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		state ProjectEnvironmentVariable
		plan  ProjectEnvironmentVariable
		want  bool
	}{
		{
			name: "unchanged version",
			state: ProjectEnvironmentVariable{
				Value:          types.StringNull(),
				ValueWOVersion: types.Int64Value(1),
			},
			plan: ProjectEnvironmentVariable{
				Value:          types.StringNull(),
				ValueWOVersion: types.Int64Value(1),
			},
			want: false,
		},
		{
			name: "changed version",
			state: ProjectEnvironmentVariable{
				Value:          types.StringNull(),
				ValueWOVersion: types.Int64Value(1),
			},
			plan: ProjectEnvironmentVariable{
				Value:          types.StringNull(),
				ValueWOVersion: types.Int64Value(2),
			},
			want: true,
		},
		{
			name: "switch from persisted value",
			state: ProjectEnvironmentVariable{
				Value:          types.StringValue("old-secret"),
				ValueWOVersion: types.Int64Null(),
			},
			plan: ProjectEnvironmentVariable{
				Value:          types.StringNull(),
				ValueWOVersion: types.Int64Null(),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldUpdateProjectEnvironmentVariableValueWO(tt.state, tt.plan); got != tt.want {
				t.Fatalf("shouldUpdateProjectEnvironmentVariableValueWO() = %t, want %t", got, tt.want)
			}
		})
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
		ValueWOVersion:       types.Int64Null(),
		TeamID:               types.StringNull(),
		ProjectID:            types.StringValue("prj_123"),
		ID:                   types.StringNull(),
		Sensitive:            types.BoolValue(true),
		Visibility:           types.StringNull(),
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
