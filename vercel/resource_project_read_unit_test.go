package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

func TestConvertResponseToProjectHandlesMissingResourceConfigAndBranchSensitiveEnv(t *testing.T) {
	ctx := context.Background()
	mainBranch := "main"
	featureBranch := "feature"
	plan := projectForReadTests()
	plan.Environment = types.SetValueMust(envVariableElemType, []attr.Value{
		(&EnvironmentItem{
			Target:               stringSet("preview"),
			CustomEnvironmentIDs: types.SetNull(types.StringType),
			GitBranch:            types.StringValue(mainBranch),
			Key:                  types.StringValue("SECRET"),
			Value:                types.StringValue("main-secret"),
			ID:                   types.StringNull(),
			Sensitive:            types.BoolValue(true),
			Comment:              types.StringNull(),
		}).toAttrValue(),
		(&EnvironmentItem{
			Target:               stringSet("preview"),
			CustomEnvironmentIDs: types.SetNull(types.StringType),
			GitBranch:            types.StringValue(featureBranch),
			Key:                  types.StringValue("SECRET"),
			Value:                types.StringValue("feature-secret"),
			ID:                   types.StringNull(),
			Sensitive:            types.BoolValue(true),
			Comment:              types.StringNull(),
		}).toAttrValue(),
	})

	result, err := convertResponseToProject(ctx, client.ProjectResponse{
		ID:     "prj_123",
		Name:   "example",
		TeamID: "team_123",
	}, plan, []client.EnvironmentVariable{
		{
			ID:        "env_main",
			Key:       "SECRET",
			Target:    []string{"preview"},
			GitBranch: &mainBranch,
			Type:      "sensitive",
		},
		{
			ID:        "env_feature",
			Key:       "SECRET",
			Target:    []string{"preview"},
			GitBranch: &featureBranch,
			Type:      "sensitive",
		},
	})
	if err != nil {
		t.Fatalf("convertResponseToProject() returned error: %v", err)
	}
	if !result.OnDemandConcurrentBuilds.IsNull() {
		t.Fatalf("OnDemandConcurrentBuilds = %v, want null", result.OnDemandConcurrentBuilds)
	}
	if !result.BuildMachineType.IsNull() {
		t.Fatalf("BuildMachineType = %v, want null", result.BuildMachineType)
	}

	var environment []EnvironmentItem
	diags := result.Environment.ElementsAs(ctx, &environment, true)
	if diags.HasError() {
		t.Fatalf("Environment.ElementsAs() returned diagnostics: %v", diags)
	}
	valuesByBranch := map[string]string{}
	for _, item := range environment {
		valuesByBranch[item.GitBranch.ValueString()] = item.Value.ValueString()
	}
	if got := valuesByBranch[mainBranch]; got != "main-secret" {
		t.Fatalf("main branch value = %q, want %q", got, "main-secret")
	}
	if got := valuesByBranch[featureBranch]; got != "feature-secret" {
		t.Fatalf("feature branch value = %q, want %q", got, "feature-secret")
	}
}

func projectForReadTests() Project {
	return Project{
		Name:                              types.StringValue("example"),
		BuildCommand:                      types.StringNull(),
		IgnoreCommand:                     types.StringNull(),
		DevCommand:                        types.StringNull(),
		Framework:                         types.StringNull(),
		InstallCommand:                    types.StringNull(),
		OutputDirectory:                   types.StringNull(),
		PreviewDeploymentSuffix:           types.StringNull(),
		PublicSource:                      types.BoolNull(),
		RootDirectory:                     types.StringNull(),
		GitRepository:                     types.ObjectNull(gitRepositoryAttrType.AttrTypes),
		VercelAuthentication:              types.ObjectNull(vercelAuthenticationAttrType.AttrTypes),
		PasswordProtection:                types.ObjectNull(passwordProtectionWithPasswordAttrType.AttrTypes),
		TrustedIps:                        types.ObjectNull(trustedIpsAttrType.AttrTypes),
		OIDCTokenConfig:                   types.ObjectNull(oidcTokenConfigAttrType.AttrTypes),
		OptionsAllowlist:                  types.ObjectNull(optionsAllowlistAttrType.AttrTypes),
		AutoExposeSystemEnvVars:           types.BoolUnknown(),
		PreviewComments:                   types.BoolNull(),
		EnablePreviewFeedback:             types.BoolNull(),
		EnableProductionFeedback:          types.BoolNull(),
		EnableAffectedProjectsDeployments: types.BoolNull(),
		PreviewDeploymentsDisabled:        types.BoolUnknown(),
		AutoAssignCustomDomains:           types.BoolValue(true),
		GitLFS:                            types.BoolUnknown(),
		FunctionFailover:                  types.BoolUnknown(),
		CustomerSuccessCodeVisibility:     types.BoolUnknown(),
		GitForkProtection:                 types.BoolValue(true),
		PrioritiseProductionBuilds:        types.BoolUnknown(),
		DirectoryListing:                  types.BoolUnknown(),
		SkewProtection:                    types.StringNull(),
		GitComments:                       types.ObjectNull(gitCommentsAttrTypes),
		GitProviderOptions:                types.ObjectNull(gitProviderOptionsAttrType.AttrTypes),
		ResourceConfig:                    types.ObjectNull(resourceConfigAttrType.AttrTypes),
		NodeVersion:                       types.StringUnknown(),
		OnDemandConcurrentBuilds:          types.BoolUnknown(),
		BuildMachineType:                  types.StringUnknown(),
		Environment:                       types.SetNull(envVariableElemType),
	}
}
