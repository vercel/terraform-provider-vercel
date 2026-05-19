package vercel

import (
	"context"
	"reflect"
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

func TestProjectTrustedSourcesToUpdateProjectRequest(t *testing.T) {
	ctx := context.Background()
	project := projectForUpdateRequestTests()
	project.TrustedSources = trustedSourcesValue()

	request, diags := project.toUpdateProjectRequest(ctx, project.Name.ValueString())
	if diags.HasError() {
		t.Fatalf("toUpdateProjectRequest() returned diagnostics: %v", diags)
	}
	if request.TrustedSources == nil {
		t.Fatal("TrustedSources = nil, want configured trusted sources")
	}

	sourceProject, ok := request.TrustedSources.Projects["prj_source"]
	if !ok {
		t.Fatalf("TrustedSources.Projects is missing prj_source: %#v", request.TrustedSources.Projects)
	}
	if sourceProject.Label == nil || *sourceProject.Label != "Source project" {
		t.Fatalf("source project label = %v, want Source project", sourceProject.Label)
	}
	if len(sourceProject.CustomAllow) != 1 {
		t.Fatalf("source project custom allow rules = %d, want 1", len(sourceProject.CustomAllow))
	}
	if !reflect.DeepEqual(sourceProject.CustomAllow[0].From.Slugs, []string{"production"}) {
		t.Fatalf("source project from slugs = %#v, want production", sourceProject.CustomAllow[0].From.Slugs)
	}
	if !reflect.DeepEqual(sourceProject.CustomAllow[0].To.Slugs, []string{"preview", "production"}) {
		t.Fatalf("source project to slugs = %#v, want preview and production", sourceProject.CustomAllow[0].To.Slugs)
	}

	providers := request.TrustedSources.OIDCProviders["https://token.actions.githubusercontent.com"]
	if len(providers) != 1 {
		t.Fatalf("trusted OIDC provider entries = %d, want 1", len(providers))
	}
	provider := providers[0]
	if provider.Label == nil || *provider.Label != "GitHub Actions" {
		t.Fatalf("trusted OIDC provider label = %v, want GitHub Actions", provider.Label)
	}
	if !reflect.DeepEqual(provider.To.Slugs, []string{"preview"}) {
		t.Fatalf("trusted OIDC provider target slugs = %#v, want preview", provider.To.Slugs)
	}
	if !reflect.DeepEqual(provider.Claims["aud"], []string{"example-audience"}) {
		t.Fatalf("trusted OIDC provider aud claim = %#v, want example-audience", provider.Claims["aud"])
	}
	assertStringSliceSet(t, provider.Claims["sub"], []string{
		"repo:vercel/example:ref:refs/heads/main",
		"repo:vercel/example:ref:refs/heads/dev",
	})
}

func TestProjectTrustedSourcesRejectsDuplicateProjects(t *testing.T) {
	ctx := context.Background()
	project := projectForUpdateRequestTests()
	project.TrustedSources = trustedSourcesDuplicateProjectValue()

	_, diags := project.toUpdateProjectRequest(ctx, project.Name.ValueString())
	if !diags.HasError() {
		t.Fatal("toUpdateProjectRequest() returned no diagnostics, want duplicate project_id error")
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
		TrustedSources:                    types.ObjectNull(trustedSourcesAttrType.AttrTypes),
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

func trustedSourcesValue() types.Object {
	return types.ObjectValueMust(trustedSourcesAttrType.AttrTypes, map[string]attr.Value{
		"projects": types.SetValueMust(trustedSourcesProjectAttrType, []attr.Value{
			types.ObjectValueMust(trustedSourcesProjectAttrType.AttrTypes, map[string]attr.Value{
				"project_id": types.StringValue("prj_source"),
				"label":      types.StringValue("Source project"),
				"custom_allow": types.SetValueMust(trustedSourcesAccessRuleAttrType, []attr.Value{
					types.ObjectValueMust(trustedSourcesAccessRuleAttrType.AttrTypes, map[string]attr.Value{
						"from": trustedSourcesEnvMatcherValue([]string{"production"}, nil),
						"to":   trustedSourcesEnvMatcherValue([]string{"preview", "production"}, nil),
					}),
				}),
			}),
		}),
		"oidc_providers": types.SetValueMust(trustedSourcesOIDCProviderAttrType, []attr.Value{
			types.ObjectValueMust(trustedSourcesOIDCProviderAttrType.AttrTypes, map[string]attr.Value{
				"issuer": types.StringValue("https://token.actions.githubusercontent.com"),
				"label":  types.StringValue("GitHub Actions"),
				"to":     trustedSourcesEnvMatcherValue([]string{"preview"}, nil),
				"claims": trustedSourcesClaimsValue(map[string][]string{
					"aud": {"example-audience"},
					"sub": {
						"repo:vercel/example:ref:refs/heads/main",
						"repo:vercel/example:ref:refs/heads/dev",
					},
				}),
			}),
		}),
	})
}

func trustedSourcesDuplicateProjectValue() types.Object {
	return types.ObjectValueMust(trustedSourcesAttrType.AttrTypes, map[string]attr.Value{
		"projects": types.SetValueMust(trustedSourcesProjectAttrType, []attr.Value{
			types.ObjectValueMust(trustedSourcesProjectAttrType.AttrTypes, map[string]attr.Value{
				"project_id":   types.StringValue("prj_source"),
				"label":        types.StringValue("Source project"),
				"custom_allow": types.SetNull(trustedSourcesAccessRuleAttrType),
			}),
			types.ObjectValueMust(trustedSourcesProjectAttrType.AttrTypes, map[string]attr.Value{
				"project_id":   types.StringValue("prj_source"),
				"label":        types.StringValue("Duplicate source project"),
				"custom_allow": types.SetNull(trustedSourcesAccessRuleAttrType),
			}),
		}),
		"oidc_providers": types.SetNull(trustedSourcesOIDCProviderAttrType),
	})
}

func trustedSourcesEnvMatcherValue(slugs []string, preset *string) types.Object {
	slugsValue := types.SetNull(types.StringType)
	if slugs != nil {
		slugValues := make([]attr.Value, 0, len(slugs))
		for _, slug := range slugs {
			slugValues = append(slugValues, types.StringValue(slug))
		}
		slugsValue = types.SetValueMust(types.StringType, slugValues)
	}

	presetValue := types.StringNull()
	if preset != nil {
		presetValue = types.StringValue(*preset)
	}

	return types.ObjectValueMust(trustedSourcesEnvMatcherAttrType.AttrTypes, map[string]attr.Value{
		"slugs":  slugsValue,
		"preset": presetValue,
	})
}

func trustedSourcesClaimsValue(claims map[string][]string) types.Map {
	values := make(map[string]attr.Value, len(claims))
	for name, claimValues := range claims {
		setValues := make([]attr.Value, 0, len(claimValues))
		for _, value := range claimValues {
			setValues = append(setValues, types.StringValue(value))
		}
		values[name] = types.SetValueMust(types.StringType, setValues)
	}
	return types.MapValueMust(trustedSourcesClaimValuesType, values)
}

func assertStringSliceSet(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	remaining := map[string]int{}
	for _, value := range want {
		remaining[value]++
	}
	for _, value := range got {
		remaining[value]--
	}
	for value, count := range remaining {
		if count != 0 {
			t.Fatalf("got %#v, want %#v; value %q count delta = %d", got, want, value, count)
		}
	}
}
