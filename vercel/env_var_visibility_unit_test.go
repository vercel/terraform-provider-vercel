package vercel

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestResolveEnvVarTypeAndVisibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		sensitive      types.Bool
		visibility     types.String
		wantType       string
		wantVisibility *string
		wantError      bool
	}{
		{
			name:           "legacy sensitive true omits visibility",
			sensitive:      types.BoolValue(true),
			visibility:     types.StringNull(),
			wantType:       "sensitive",
			wantVisibility: nil,
		},
		{
			name:           "legacy sensitive false omits visibility",
			sensitive:      types.BoolValue(false),
			visibility:     types.StringNull(),
			wantType:       "encrypted",
			wantVisibility: nil,
		},
		{
			name:           "explicit secret visibility",
			sensitive:      types.BoolValue(true),
			visibility:     types.StringValue("secret"),
			wantType:       "sensitive",
			wantVisibility: strPtr("secret"),
		},
		{
			name:           "explicit config visibility",
			sensitive:      types.BoolValue(false),
			visibility:     types.StringValue("config"),
			wantType:       "encrypted",
			wantVisibility: strPtr("config"),
		},
		{
			name:       "conflicting sensitive true and config visibility",
			sensitive:  types.BoolValue(true),
			visibility: types.StringValue("config"),
			wantError:  true,
		},
		{
			name:       "conflicting sensitive false and secret visibility",
			sensitive:  types.BoolValue(false),
			visibility: types.StringValue("secret"),
			wantError:  true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotType, gotVisibility, diags := resolveEnvVarTypeAndVisibility(tt.sensitive, tt.visibility)
			if tt.wantError {
				if !diags.HasError() {
					t.Fatal("expected error, got none")
				}
				return
			}
			if diags.HasError() {
				t.Fatalf("unexpected error: %s", diags)
			}
			if gotType != tt.wantType {
				t.Fatalf("type = %q, want %q", gotType, tt.wantType)
			}
			if tt.wantVisibility == nil {
				if gotVisibility != nil {
					t.Fatalf("visibility = %v, want nil", gotVisibility)
				}
				return
			}
			if gotVisibility == nil || *gotVisibility != *tt.wantVisibility {
				t.Fatalf("visibility = %v, want %v", gotVisibility, tt.wantVisibility)
			}
		})
	}
}

func TestEnvVarVisibilityFromResponse(t *testing.T) {
	t.Parallel()

	if got := envVarVisibilityFromResponse("sensitive", nil); got.ValueString() != "secret" {
		t.Fatalf("got %q, want secret", got.ValueString())
	}
	if got := envVarVisibilityFromResponse("encrypted", nil); got.ValueString() != "config" {
		t.Fatalf("got %q, want config", got.ValueString())
	}
	if got := envVarVisibilityFromResponse("encrypted", strPtr("secret")); got.ValueString() != "secret" {
		t.Fatalf("got %q, want secret", got.ValueString())
	}
}

func strPtr(s string) *string {
	return &s
}
