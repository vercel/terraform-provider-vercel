package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestToClientResourceConfig_BuildMachineType(t *testing.T) {
	tests := []struct {
		name              string
		buildMachineType  types.String
		wantBuildMachine  *string
		wantSelection     *string
	}{
		{
			name:             "null is omitted",
			buildMachineType: types.StringNull(),
		},
		// Regression: an adopted project whose API response had no
		// buildMachineType previously landed in state as
		// types.StringValue("") and was forwarded back to the API on
		// update, which rejects an explicit empty string.
		{
			name:             "empty string is omitted",
			buildMachineType: types.StringValue(""),
		},
		{
			name:             "unknown is omitted",
			buildMachineType: types.StringUnknown(),
		},
		{
			name:             "enhanced is sent as buildMachineType",
			buildMachineType: types.StringValue("enhanced"),
			wantBuildMachine: stringPtr("enhanced"),
		},
		{
			name:             "elastic is sent as buildMachineSelection",
			buildMachineType: types.StringValue("elastic"),
			wantSelection:    stringPtr("elastic"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var rc *ResourceConfig
			got := rc.toClientResourceConfig(
				context.Background(),
				types.BoolNull(),
				tc.buildMachineType,
				types.StringNull(),
			)

			if got == nil {
				if tc.wantBuildMachine != nil || tc.wantSelection != nil {
					t.Fatalf("expected resource config, got nil")
				}
				return
			}
			if !stringPtrEq(got.BuildMachineType, tc.wantBuildMachine) {
				t.Errorf("BuildMachineType: got %v, want %v", deref(got.BuildMachineType), deref(tc.wantBuildMachine))
			}
			if !stringPtrEq(got.BuildMachineSelection, tc.wantSelection) {
				t.Errorf("BuildMachineSelection: got %v, want %v", deref(got.BuildMachineSelection), deref(tc.wantSelection))
			}
		})
	}
}

func stringPtr(v string) *string { return &v }

func stringPtrEq(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func deref(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}
