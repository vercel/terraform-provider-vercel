package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

func TestProjectEnvironmentVariablesModifyPlanSkipsPolicyValidationForExistingVariables(t *testing.T) {
	ctx := context.Background()
	policy := "on"
	res := &projectEnvironmentVariablesResource{
		client: client.New("").WithTeam(client.Team{
			SensitiveEnvironmentVariablePolicy: &policy,
		}),
	}

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	configEnv := EnvironmentItem{
		Target:               stringSet("production"),
		CustomEnvironmentIDs: types.SetNull(types.StringType),
		GitBranch:            types.StringNull(),
		Key:                  types.StringValue("EXAMPLE"),
		Value:                types.StringValue("value"),
		ID:                   types.StringNull(),
		Sensitive:            types.BoolValue(false),
		Comment:              types.StringNull(),
	}

	planEnv := configEnv
	planEnv.ID = types.StringValue("env_123")

	config := ProjectEnvironmentVariables{
		ID:        types.StringNull(),
		TeamID:    types.StringNull(),
		ProjectID: types.StringValue("prj_123"),
		Variables: types.SetValueMust(envVariableElemType, []attr.Value{environmentItemAttrValue(configEnv)}),
	}

	plan := config
	plan.Variables = types.SetValueMust(envVariableElemType, []attr.Value{environmentItemAttrValue(planEnv)})

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

func TestProjectEnvironmentVariablesModifyPlanUsesPlannedDevelopmentTarget(t *testing.T) {
	ctx := context.Background()
	res := &projectEnvironmentVariablesResource{}

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	configEnv := EnvironmentItem{
		Target:               types.SetNull(types.StringType),
		CustomEnvironmentIDs: types.SetNull(types.StringType),
		GitBranch:            types.StringNull(),
		Key:                  types.StringValue("EXAMPLE"),
		Value:                types.StringValue("value"),
		ID:                   types.StringNull(),
		Sensitive:            types.BoolValue(true),
		Comment:              types.StringNull(),
	}

	planEnv := configEnv
	planEnv.Target = stringSet("development")
	planEnv.ID = types.StringValue("env_123")
	planEnv.Sensitive = types.BoolValue(true)

	config := ProjectEnvironmentVariables{
		ID:        types.StringNull(),
		TeamID:    types.StringNull(),
		ProjectID: types.StringValue("prj_123"),
		Variables: types.SetValueMust(envVariableElemType, []attr.Value{environmentItemAttrValue(configEnv)}),
	}

	plan := config
	plan.Variables = types.SetValueMust(envVariableElemType, []attr.Value{environmentItemAttrValue(planEnv)})

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

func environmentItemAttrValue(e EnvironmentItem) attr.Value {
	return types.ObjectValueMust(envVariableElemType.AttrTypes, map[string]attr.Value{
		"id":                     e.ID,
		"key":                    e.Key,
		"value":                  e.Value,
		"target":                 e.Target,
		"custom_environment_ids": e.CustomEnvironmentIDs,
		"git_branch":             e.GitBranch,
		"sensitive":              e.Sensitive,
		"comment":                e.Comment,
	})
}
