package vercel

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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

func TestConvertResponseToProjectPreservesConfiguredPublicSourceWhenResponseOmitsIt(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name            string
		configuredValue types.Bool
		wantNull        bool
		wantStateValue  bool
	}{
		{
			name:            "configured true",
			configuredValue: types.BoolValue(true),
			wantStateValue:  true,
		},
		{
			name:            "configured false",
			configuredValue: types.BoolValue(false),
			wantStateValue:  false,
		},
		{
			name:            "unset",
			configuredValue: types.BoolNull(),
			wantNull:        true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			plan := projectForReadTests()
			plan.PublicSource = tt.configuredValue

			result, err := convertResponseToProject(ctx, client.ProjectResponse{
				ID:     "prj_123",
				Name:   "example",
				TeamID: "team_123",
			}, plan, nil)
			if err != nil {
				t.Fatalf("convertResponseToProject() returned error: %v", err)
			}

			if result.PublicSource.IsNull() != tt.wantNull {
				t.Fatalf("PublicSource null = %t, want %t", result.PublicSource.IsNull(), tt.wantNull)
			}
			if !tt.wantNull && result.PublicSource.ValueBool() != tt.wantStateValue {
				t.Fatalf("PublicSource = %t, want %t", result.PublicSource.ValueBool(), tt.wantStateValue)
			}
		})
	}
}

func TestConvertResponseToProjectTrustedSources(t *testing.T) {
	ctx := context.Background()
	projectLabel := "Source project"
	providerLabel := "GitHub Actions"
	result, err := convertResponseToProject(ctx, client.ProjectResponse{
		ID:     "prj_123",
		Name:   "example",
		TeamID: "team_123",
		TrustedSources: &client.TrustedSources{
			Projects: map[string]client.TrustedSourcesProject{
				"prj_source": {
					Label: &projectLabel,
					CustomAllow: []client.TrustedSourcesAccessRule{
						{
							From: client.TrustedSourcesEnvMatcher{Slugs: []string{"production"}},
							To:   client.TrustedSourcesEnvMatcher{Slugs: []string{"preview", "production"}},
						},
					},
				},
			},
			OIDCProviders: map[string][]client.TrustedSourcesOIDCProvider{
				"https://token.actions.githubusercontent.com": {
					{
						TrustedSourcesTargetAccess: client.TrustedSourcesTargetAccess{
							To: client.TrustedSourcesEnvMatcher{Slugs: []string{"preview"}},
						},
						Label: &providerLabel,
						Claims: client.TrustedSourcesClaims{
							"aud": {"example-audience"},
							"sub": {
								"repo:vercel/example:ref:refs/heads/main",
								"repo:vercel/example:ref:refs/heads/dev",
							},
						},
					},
				},
			},
		},
	}, projectForReadTests(), nil)
	if err != nil {
		t.Fatalf("convertResponseToProject() returned error: %v", err)
	}

	var trustedSources TrustedSources
	diags := result.TrustedSources.As(ctx, &trustedSources, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
	if diags.HasError() {
		t.Fatalf("TrustedSources.As() returned diagnostics: %v", diags)
	}

	var projects []TrustedSourcesProject
	diags = trustedSources.Projects.ElementsAs(ctx, &projects, false)
	if diags.HasError() {
		t.Fatalf("TrustedSources.Projects.ElementsAs() returned diagnostics: %v", diags)
	}
	if len(projects) != 1 {
		t.Fatalf("trusted source projects = %d, want 1", len(projects))
	}
	if projects[0].ProjectID.ValueString() != "prj_source" {
		t.Fatalf("trusted source project ID = %q, want prj_source", projects[0].ProjectID.ValueString())
	}
	if projects[0].Label.ValueString() != projectLabel {
		t.Fatalf("trusted source project label = %q, want %q", projects[0].Label.ValueString(), projectLabel)
	}
	var rules []TrustedSourcesAccessRule
	diags = projects[0].CustomAllow.ElementsAs(ctx, &rules, false)
	if diags.HasError() {
		t.Fatalf("TrustedSourcesProject.CustomAllow.ElementsAs() returned diagnostics: %v", diags)
	}
	if len(rules) != 1 {
		t.Fatalf("trusted source project custom allow rules = %d, want 1", len(rules))
	}
	from := trustedSourcesEnvMatcherFromObject(t, ctx, rules[0].From)
	to := trustedSourcesEnvMatcherFromObject(t, ctx, rules[0].To)
	if !reflect.DeepEqual(trustedSourcesSlugValues(t, ctx, from.Slugs), []string{"production"}) {
		t.Fatalf("trusted source project from slugs = %#v, want production", trustedSourcesSlugValues(t, ctx, from.Slugs))
	}
	assertStringSliceSet(t, trustedSourcesSlugValues(t, ctx, to.Slugs), []string{"preview", "production"})

	var externalSources []TrustedSourcesExternalSource
	diags = trustedSources.ExternalSources.ElementsAs(ctx, &externalSources, false)
	if diags.HasError() {
		t.Fatalf("TrustedSources.ExternalSources.ElementsAs() returned diagnostics: %v", diags)
	}
	if len(externalSources) != 1 {
		t.Fatalf("trusted external source entries = %d, want 1", len(externalSources))
	}
	if externalSources[0].Issuer.ValueString() != "https://token.actions.githubusercontent.com" {
		t.Fatalf("trusted external source issuer = %q, want GitHub Actions issuer", externalSources[0].Issuer.ValueString())
	}
	if externalSources[0].Label.ValueString() != providerLabel {
		t.Fatalf("trusted external source label = %q, want %q", externalSources[0].Label.ValueString(), providerLabel)
	}
	externalSourceTo := trustedSourcesEnvMatcherFromObject(t, ctx, externalSources[0].To)
	if !reflect.DeepEqual(trustedSourcesSlugValues(t, ctx, externalSourceTo.Slugs), []string{"preview"}) {
		t.Fatalf("trusted external source target slugs = %#v, want preview", trustedSourcesSlugValues(t, ctx, externalSourceTo.Slugs))
	}
	claims := trustedSourcesClaimsFromMap(t, ctx, externalSources[0].Claims)
	if !reflect.DeepEqual(claims["aud"], []string{"example-audience"}) {
		t.Fatalf("trusted external source aud claim = %#v, want example-audience", claims["aud"])
	}
	assertStringSliceSet(t, claims["sub"], []string{
		"repo:vercel/example:ref:refs/heads/main",
		"repo:vercel/example:ref:refs/heads/dev",
	})
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

func trustedSourcesEnvMatcherFromObject(t *testing.T, ctx context.Context, value types.Object) TrustedSourcesEnvMatcher {
	t.Helper()
	var matcher TrustedSourcesEnvMatcher
	diags := value.As(ctx, &matcher, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
	if diags.HasError() {
		t.Fatalf("trusted sources env matcher As() returned diagnostics: %v", diags)
	}
	return matcher
}

func trustedSourcesSlugValues(t *testing.T, ctx context.Context, value types.Set) []string {
	t.Helper()
	var slugs []string
	diags := value.ElementsAs(ctx, &slugs, false)
	if diags.HasError() {
		t.Fatalf("trusted sources slugs ElementsAs() returned diagnostics: %v", diags)
	}
	return slugs
}

func trustedSourcesClaimsFromMap(t *testing.T, ctx context.Context, value types.Map) map[string][]string {
	t.Helper()
	claims := map[string][]string{}
	for name, claimValues := range value.Elements() {
		valuesSet, ok := claimValues.(types.Set)
		if !ok {
			t.Fatalf("claim %q value type = %T, want types.Set", name, claimValues)
		}
		var values []string
		diags := valuesSet.ElementsAs(ctx, &values, false)
		if diags.HasError() {
			t.Fatalf("claim %q ElementsAs() returned diagnostics: %v", name, diags)
		}
		claims[name] = values
	}
	return claims
}
