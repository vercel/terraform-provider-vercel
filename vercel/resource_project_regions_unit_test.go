package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

func TestConvertResponseToProjectCanonicalizesLegacyServerlessRegion(t *testing.T) {
	region := "sfo1"
	result, err := convertResponseToProject(context.Background(), client.ProjectResponse{
		ID:                       "prj_123",
		Name:                     "example",
		ServerlessFunctionRegion: &region,
	}, nullProject, nil)
	if err != nil {
		t.Fatalf("convertResponseToProject() returned error: %v", err)
	}

	if !result.ServerlessFunctionRegion.IsNull() {
		t.Fatalf("ServerlessFunctionRegion = %v, want null for unconfigured deprecated field", result.ServerlessFunctionRegion)
	}

	assertProjectFunctionDefaultRegions(t, result, []string{"sfo1"})
}

func TestConvertResponseToProjectPrefersResourceConfigRegions(t *testing.T) {
	region := "sfo1"
	result, err := convertResponseToProject(context.Background(), client.ProjectResponse{
		ID:                       "prj_123",
		Name:                     "example",
		ServerlessFunctionRegion: &region,
		ResourceConfig: &client.ResourceConfigResponse{
			FunctionDefaultRegions: []string{"iad1", "fra1"},
		},
	}, nullProject, nil)
	if err != nil {
		t.Fatalf("convertResponseToProject() returned error: %v", err)
	}

	assertProjectFunctionDefaultRegions(t, result, []string{"iad1", "fra1"})
}

func TestConvertResponseToProjectPreservesConfiguredDeprecatedRegion(t *testing.T) {
	region := "sfo1"
	plan := nullProject
	plan.ServerlessFunctionRegion = types.StringValue(region)

	result, err := convertResponseToProject(context.Background(), client.ProjectResponse{
		ID:                       "prj_123",
		Name:                     "example",
		ServerlessFunctionRegion: &region,
	}, plan, nil)
	if err != nil {
		t.Fatalf("convertResponseToProject() returned error: %v", err)
	}

	if got := result.ServerlessFunctionRegion.ValueString(); got != region {
		t.Fatalf("ServerlessFunctionRegion = %q, want %q", got, region)
	}
	assertProjectFunctionDefaultRegions(t, result, []string{"sfo1"})
}

func assertProjectFunctionDefaultRegions(t *testing.T, project Project, want []string) {
	t.Helper()

	resourceConfig, diags := project.resourceConfig(context.Background())
	if diags.HasError() {
		t.Fatalf("resourceConfig() returned diagnostics: %v", diags)
	}
	if resourceConfig == nil {
		t.Fatal("resourceConfig() = nil, want populated resource config")
	}

	var got []string
	diags = resourceConfig.FunctionDefaultRegions.ElementsAs(context.Background(), &got, false)
	if diags.HasError() {
		t.Fatalf("FunctionDefaultRegions.ElementsAs() returned diagnostics: %v", diags)
	}
	gotByRegion := make(map[string]struct{}, len(got))
	for _, region := range got {
		gotByRegion[region] = struct{}{}
	}
	if len(gotByRegion) != len(want) {
		t.Fatalf("function_default_regions = %v, want %v", got, want)
	}
	for _, region := range want {
		if _, ok := gotByRegion[region]; !ok {
			t.Fatalf("function_default_regions = %v, want %v", got, want)
		}
	}
}
