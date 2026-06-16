package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v4/client"
)

// public_source is deprecated: it is no longer sent to or returned by the
// Vercel API. A configured value must still be preserved so it does not
// produce a perpetual diff.
func TestConvertResponseToProjectPreservesConfiguredPublicSource(t *testing.T) {
	plan := nullProject
	plan.PublicSource = types.BoolValue(true)

	result, err := convertResponseToProject(context.Background(), client.ProjectResponse{
		ID:   "prj_123",
		Name: "example",
		// API no longer returns publicSource.
	}, plan, nil)
	if err != nil {
		t.Fatalf("convertResponseToProject() returned error: %v", err)
	}

	if got := result.PublicSource; got.IsNull() || !got.ValueBool() {
		t.Fatalf("PublicSource = %v, want true (configured value preserved)", got)
	}
}

func TestConvertResponseToProjectLeavesUnconfiguredPublicSourceNull(t *testing.T) {
	result, err := convertResponseToProject(context.Background(), client.ProjectResponse{
		ID:   "prj_123",
		Name: "example",
	}, nullProject, nil)
	if err != nil {
		t.Fatalf("convertResponseToProject() returned error: %v", err)
	}

	if !result.PublicSource.IsNull() {
		t.Fatalf("PublicSource = %v, want null for unconfigured deprecated field", result.PublicSource)
	}
}
