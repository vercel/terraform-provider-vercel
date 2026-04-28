package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The Vercel API treats `buildMachineType: "elastic"` as the trigger for
// elastic mode and ignores `buildMachineSelection` from the request body.
// Earlier versions of the provider routed the "elastic" value through
// `buildMachineSelection`, which the API silently dropped — leaving the
// project unchanged and triggering "Provider produced inconsistent result
// after apply" on the next refresh.
func TestToClientResourceConfigSendsElasticAsBuildMachineType(t *testing.T) {
	for _, tc := range []struct {
		name             string
		buildMachineType types.String
		expected         string
	}{
		{name: "elastic", buildMachineType: types.StringValue("elastic"), expected: "elastic"},
		{name: "enhanced", buildMachineType: types.StringValue("enhanced"), expected: "enhanced"},
		{name: "turbo", buildMachineType: types.StringValue("turbo"), expected: "turbo"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var rc *ResourceConfig
			got := rc.toClientResourceConfig(
				context.Background(),
				types.BoolNull(),
				tc.buildMachineType,
				types.StringNull(),
			)
			if got == nil {
				t.Fatal("toClientResourceConfig returned nil, want non-nil")
			}
			if got.BuildMachineType == nil {
				t.Fatal("BuildMachineType is nil, want a pointer to a value")
			}
			if *got.BuildMachineType != tc.expected {
				t.Errorf("BuildMachineType = %q, want %q", *got.BuildMachineType, tc.expected)
			}
			if got.BuildMachineSelection != nil {
				t.Errorf("BuildMachineSelection = %q, want nil (the API derives selection from buildMachineType)", *got.BuildMachineSelection)
			}
		})
	}
}
