package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestShouldValidateSensitiveEnvironmentVariablePolicy(t *testing.T) {
	tests := []struct {
		name                         string
		target                       types.Set
		customEnvironmentIDs         types.Set
		targetsAllCustomEnvironments bool
		explicitlyNonSensitive       bool
		id                           types.String
		want                         bool
	}{
		{
			name:                         "computed sensitive skips validation",
			target:                       stringSet("production"),
			customEnvironmentIDs:         types.SetNull(types.StringType),
			targetsAllCustomEnvironments: false,
			explicitlyNonSensitive:       false,
			id:                           types.StringNull(),
			want:                         false,
		},
		{
			name:                         "existing resource skips validation",
			target:                       stringSet("production"),
			customEnvironmentIDs:         types.SetNull(types.StringType),
			targetsAllCustomEnvironments: false,
			explicitlyNonSensitive:       true,
			id:                           types.StringValue("env_123"),
			want:                         false,
		},
		{
			name:                         "production validates when explicitly non-sensitive",
			target:                       stringSet("production"),
			customEnvironmentIDs:         types.SetNull(types.StringType),
			targetsAllCustomEnvironments: false,
			explicitlyNonSensitive:       true,
			id:                           types.StringNull(),
			want:                         true,
		},
		{
			name:                         "development only with unknown custom environments skips validation",
			target:                       stringSet("development"),
			customEnvironmentIDs:         types.SetUnknown(types.StringType),
			targetsAllCustomEnvironments: false,
			explicitlyNonSensitive:       true,
			id:                           types.StringNull(),
			want:                         false,
		},
		{
			name:                         "development and preview validates",
			target:                       stringSet("development", "preview"),
			customEnvironmentIDs:         types.SetNull(types.StringType),
			targetsAllCustomEnvironments: false,
			explicitlyNonSensitive:       true,
			id:                           types.StringNull(),
			want:                         true,
		},
		{
			name:                         "custom environment only validates",
			target:                       types.SetNull(types.StringType),
			customEnvironmentIDs:         stringSet("ce_123"),
			targetsAllCustomEnvironments: false,
			explicitlyNonSensitive:       true,
			id:                           types.StringNull(),
			want:                         true,
		},
		{
			name:                         "development with custom environments validates",
			target:                       stringSet("development"),
			customEnvironmentIDs:         stringSet("ce_123"),
			targetsAllCustomEnvironments: false,
			explicitlyNonSensitive:       true,
			id:                           types.StringNull(),
			want:                         true,
		},
		{
			name:                         "development with apply all custom environments validates",
			target:                       stringSet("development"),
			customEnvironmentIDs:         types.SetNull(types.StringType),
			targetsAllCustomEnvironments: true,
			explicitlyNonSensitive:       true,
			id:                           types.StringNull(),
			want:                         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, diags := shouldValidateSensitiveEnvironmentVariablePolicy(
				context.Background(),
				tt.target,
				tt.customEnvironmentIDs,
				tt.targetsAllCustomEnvironments,
				tt.explicitlyNonSensitive,
				tt.id,
			)
			if diags.HasError() {
				t.Fatalf("shouldValidateSensitiveEnvironmentVariablePolicy() returned diagnostics: %v", diags)
			}
			if got != tt.want {
				t.Fatalf("shouldValidateSensitiveEnvironmentVariablePolicy() = %t, want %t", got, tt.want)
			}
		})
	}
}
