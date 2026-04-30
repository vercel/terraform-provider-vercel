package vercel

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestProjectRequiresUpdateAfterCreationOnlyForConfiguredFields(t *testing.T) {
	project := projectForUpdateRequestTests()
	if project.RequiresUpdateAfterCreation() {
		t.Fatal("RequiresUpdateAfterCreation() = true, want false for unset fields")
	}

	project.GitComments = types.ObjectValueMust(gitCommentsAttrTypes, map[string]attr.Value{
		"on_pull_request": types.BoolValue(true),
		"on_commit":       types.BoolValue(false),
	})
	if !project.RequiresUpdateAfterCreation() {
		t.Fatal("RequiresUpdateAfterCreation() = false, want true for configured git_comments")
	}
}

func projectForUpdateRequestTests() Project {
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
