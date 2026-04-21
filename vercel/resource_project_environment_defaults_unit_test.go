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

func TestProjectEnvironmentVariablesResourceSchemaRequiresSensitive(t *testing.T) {
	res := newProjectEnvironmentVariablesResource()

	resp := &resource.SchemaResponse{}
	res.Schema(context.Background(), resource.SchemaRequest{}, resp)

	variablesAttr, ok := resp.Schema.Attributes["variables"].(schema.SetNestedAttribute)
	if !ok {
		t.Fatalf("variables attribute has unexpected type: %T", resp.Schema.Attributes["variables"])
	}

	sensitiveAttr, ok := variablesAttr.NestedObject.Attributes["sensitive"].(schema.BoolAttribute)
	if !ok {
		t.Fatalf("variables.sensitive attribute has unexpected type: %T", variablesAttr.NestedObject.Attributes["sensitive"])
	}

	assertBoolRequired(t, sensitiveAttr, "variables.sensitive")
}

func TestProjectResourceEnvironmentSchemaRequiresSensitive(t *testing.T) {
	res := newProjectResource()

	resp := &resource.SchemaResponse{}
	res.Schema(context.Background(), resource.SchemaRequest{}, resp)

	environmentAttr, ok := resp.Schema.Attributes["environment"].(schema.SetNestedAttribute)
	if !ok {
		t.Fatalf("environment attribute has unexpected type: %T", resp.Schema.Attributes["environment"])
	}

	sensitiveAttr, ok := environmentAttr.NestedObject.Attributes["sensitive"].(schema.BoolAttribute)
	if !ok {
		t.Fatalf("environment.sensitive attribute has unexpected type: %T", environmentAttr.NestedObject.Attributes["sensitive"])
	}

	assertBoolRequired(t, sensitiveAttr, "environment.sensitive")
}

func TestEnvironmentItemSensitiveSemantics(t *testing.T) {
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
			env := EnvironmentItem{Sensitive: tt.sensitive}

			if got := env.isExplicitlyNonSensitive(); got != tt.wantExplicitlyNonSensitive {
				t.Fatalf("isExplicitlyNonSensitive() = %t, want %t", got, tt.wantExplicitlyNonSensitive)
			}

			if got := env.isSensitive(); got != tt.wantSensitive {
				t.Fatalf("isSensitive() = %t, want %t", got, tt.wantSensitive)
			}
		})
	}
}

func TestEnvironmentItemHasTarget(t *testing.T) {
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
			env := EnvironmentItem{Target: tt.target}

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

func TestEnvironmentItemToEnvironmentVariableRequestTreatsUnsetSensitiveAsSensitive(t *testing.T) {
	env := EnvironmentItem{
		Target:               types.SetNull(types.StringType),
		CustomEnvironmentIDs: stringSet("ce_123"),
		Key:                  types.StringValue("EXAMPLE"),
		Value:                types.StringValue("value"),
		Sensitive:            types.BoolNull(),
	}

	req, diags := env.toEnvironmentVariableRequest(context.Background())
	if diags.HasError() {
		t.Fatalf("toEnvironmentVariableRequest() returned diagnostics: %v", diags)
	}

	if req.Type != "sensitive" {
		t.Fatalf("toEnvironmentVariableRequest().Type = %q, want %q", req.Type, "sensitive")
	}

	if len(req.CustomEnvironmentIDs) != 1 || req.CustomEnvironmentIDs[0] != "ce_123" {
		t.Fatalf("toEnvironmentVariableRequest().CustomEnvironmentIDs = %v, want [ce_123]", req.CustomEnvironmentIDs)
	}
}

func TestProjectResourceModifyPlanSkipsPolicyValidationForExistingInlineEnvironmentVariable(t *testing.T) {
	ctx := context.Background()
	policy := "on"
	res := &projectResource{
		client: client.New("").WithTeam(client.Team{
			SensitiveEnvironmentVariablePolicy: &policy,
		}),
	}

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	configEnv := EnvironmentItem{
		Target:               stringSet("production"),
		CustomEnvironmentIDs: types.SetNull(types.StringType),
		Key:                  types.StringValue("EXAMPLE"),
		Value:                types.StringValue("value"),
		ID:                   types.StringNull(),
		Sensitive:            types.BoolValue(false),
	}
	planEnv := configEnv
	planEnv.ID = types.StringValue("env_123")

	config := Project{
		Name:                 types.StringValue("example"),
		TeamID:               types.StringNull(),
		Environment:          types.SetValueMust(envVariableElemType, []attr.Value{configEnv.toAttrValue()}),
		GitRepository:        types.ObjectNull(gitRepositoryAttrType.AttrTypes),
		VercelAuthentication: types.ObjectNull(vercelAuthenticationAttrType.AttrTypes),
		PasswordProtection:   types.ObjectNull(passwordProtectionWithPasswordAttrType.AttrTypes),
		TrustedIps:           types.ObjectNull(trustedIpsAttrType.AttrTypes),
		OIDCTokenConfig:      types.ObjectNull(oidcTokenConfigAttrType.AttrTypes),
		OptionsAllowlist:     types.ObjectNull(optionsAllowlistAttrType.AttrTypes),
		GitComments:          types.ObjectNull(gitCommentsAttrTypes),
		GitProviderOptions:   types.ObjectNull(gitProviderOptionsAttrType.AttrTypes),
		ResourceConfig:       types.ObjectNull(resourceConfigAttrType.AttrTypes),
	}
	plan := config
	plan.Environment = types.SetValueMust(envVariableElemType, []attr.Value{planEnv.toAttrValue()})

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

func assertBoolRequired(t *testing.T, attr schema.BoolAttribute, label string) {
	t.Helper()

	if !attr.Required {
		t.Fatalf("%s should be required", label)
	}
	if attr.Optional {
		t.Fatalf("%s should not be optional", label)
	}
	if attr.Computed {
		t.Fatalf("%s should not be computed", label)
	}
	if attr.Default != nil {
		t.Fatalf("%s should not have a default", label)
	}
}
