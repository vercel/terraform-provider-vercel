package vercel

import (
	"context"
	"encoding/json"
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

func TestProjectUpdateRequestOmitsUnknownPostCreateFields(t *testing.T) {
	project := projectForUpdateRequestTests()

	req, diags := project.toUpdateProjectRequest(context.Background(), project.Name.ValueString())
	if diags.HasError() {
		t.Fatalf("toUpdateProjectRequest() returned diagnostics: %v", diags)
	}

	payload := map[string]any{}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	for _, field := range []string{
		"autoExposeSystemEnvs",
		"previewDeploymentsDisabled",
		"gitLFS",
		"serverlessFunctionZeroConfigFailover",
		"customerSupportCodeVisibility",
		"productionDeploymentsFastLane",
		"directoryListing",
		"skewProtectionMaxAge",
	} {
		if _, ok := payload[field]; ok {
			t.Fatalf("field %q was included for an unknown/unset value: %s", field, b)
		}
	}
}

func TestProjectUpdateRequestSendsKnownFalsePostCreateFields(t *testing.T) {
	project := projectForUpdateRequestTests()
	project.GitLFS = types.BoolValue(false)

	req, diags := project.toUpdateProjectRequest(context.Background(), project.Name.ValueString())
	if diags.HasError() {
		t.Fatalf("toUpdateProjectRequest() returned diagnostics: %v", diags)
	}

	payload := map[string]any{}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	got, ok := payload["gitLFS"].(bool)
	if !ok {
		t.Fatalf("gitLFS was not included as a bool: %s", b)
	}
	if got {
		t.Fatalf("gitLFS = true, want false")
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
