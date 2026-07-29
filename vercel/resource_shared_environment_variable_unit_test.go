package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSharedEnvironmentVariableResourceSchemaRequiresSensitive(t *testing.T) {
	res := newSharedEnvironmentVariableResource()

	resp := &resource.SchemaResponse{}
	res.Schema(context.Background(), resource.SchemaRequest{}, resp)

	sensitiveAttr, ok := resp.Schema.Attributes["sensitive"].(schema.BoolAttribute)
	if !ok {
		t.Fatalf("sensitive attribute has unexpected type: %T", resp.Schema.Attributes["sensitive"])
	}

	assertBoolRequired(t, sensitiveAttr, "sensitive")
}

func TestConvertResponseDoesNotTrackProjectsWhenProjectIDsAreUnconfigured(t *testing.T) {
	projectIDs := types.SetNull(types.StringType)
	result := convertResponseToSharedEnvironmentVariable(client.SharedEnvironmentVariableResponse{
		ID:         "env_123",
		Key:        "EXAMPLE",
		ProjectIDs: []string{"prj_123"},
	}, types.StringValue("value"), types.Int64Null(), projectIDs)

	if !result.ProjectIDs.IsNull() {
		t.Errorf("ProjectIDs = %#v, want null", result.ProjectIDs)
	}
}

func TestConvertResponseTracksConfiguredProjectIDs(t *testing.T) {
	projectIDs := stringSet("prj_123")
	result := convertResponseToSharedEnvironmentVariable(client.SharedEnvironmentVariableResponse{
		ID:         "env_123",
		Key:        "EXAMPLE",
		ProjectIDs: []string{"prj_456"},
	}, types.StringValue("value"), types.Int64Null(), projectIDs)

	var got []string
	diags := result.ProjectIDs.ElementsAs(context.Background(), &got, false)
	if diags.HasError() {
		t.Fatalf("ProjectIDs.ElementsAs() returned diagnostics: %v", diags)
	}
	if len(got) != 1 || got[0] != "prj_123" {
		t.Errorf("ProjectIDs = %#v, want []string{\"prj_123\"}", got)
	}
}

func TestConvertResponseToSharedEnvironmentVariableKeepsWriteOnlyValueNull(t *testing.T) {
	t.Parallel()

	result := convertResponseToSharedEnvironmentVariable(
		client.SharedEnvironmentVariableResponse{
			ID:      "env_123",
			Key:     "SECRET",
			Value:   "bar-wo",
			Target:  []string{"production"},
			Type:    "sensitive",
			Comment: "test comment",
		},
		types.StringNull(),
		types.Int64Value(2),
		types.SetNull(types.StringType),
	)

	if !result.Value.IsNull() {
		t.Fatalf("Value = %v, want null", result.Value)
	}
	if !result.ValueWO.IsNull() {
		t.Fatalf("ValueWO = %v, want null", result.ValueWO)
	}
	if got := result.ValueWOVersion.ValueInt64(); got != 2 {
		t.Fatalf("ValueWOVersion = %d, want 2", got)
	}
}

func TestSharedEnvironmentVariableUpdateRequestOmitsUnchangedWriteOnlyValue(t *testing.T) {
	t.Parallel()

	env := SharedEnvironmentVariable{
		Target:    stringSet("production"),
		Value:     types.StringNull(),
		Sensitive: types.BoolValue(true),
	}

	request, ok := env.toUpdateSharedEnvironmentVariableRequest(context.Background(), nil, types.StringNull())
	if !ok {
		t.Fatal("toUpdateSharedEnvironmentVariableRequest() returned !ok")
	}
	if request.Value != nil {
		t.Fatalf("Value = %q, want nil", *request.Value)
	}
}

func TestSharedEnvironmentVariableUpdateRequestIncludesChangedWriteOnlyValue(t *testing.T) {
	t.Parallel()

	env := SharedEnvironmentVariable{
		Target:    stringSet("production"),
		Value:     types.StringNull(),
		Sensitive: types.BoolValue(true),
	}

	request, ok := env.toUpdateSharedEnvironmentVariableRequest(context.Background(), nil, types.StringValue("new-secret"))
	if !ok {
		t.Fatal("toUpdateSharedEnvironmentVariableRequest() returned !ok")
	}
	if request.Value == nil || *request.Value != "new-secret" {
		t.Fatalf("Value = %v, want new-secret", request.Value)
	}
}

func TestShouldUpdateSharedEnvironmentVariableValueWO(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		state SharedEnvironmentVariable
		plan  SharedEnvironmentVariable
		want  bool
	}{
		{
			name: "unchanged version",
			state: SharedEnvironmentVariable{
				Value:          types.StringNull(),
				ValueWOVersion: types.Int64Value(1),
			},
			plan: SharedEnvironmentVariable{
				Value:          types.StringNull(),
				ValueWOVersion: types.Int64Value(1),
			},
			want: false,
		},
		{
			name: "changed version",
			state: SharedEnvironmentVariable{
				Value:          types.StringNull(),
				ValueWOVersion: types.Int64Value(1),
			},
			plan: SharedEnvironmentVariable{
				Value:          types.StringNull(),
				ValueWOVersion: types.Int64Value(2),
			},
			want: true,
		},
		{
			name: "switch from persisted value",
			state: SharedEnvironmentVariable{
				Value:          types.StringValue("old-secret"),
				ValueWOVersion: types.Int64Null(),
			},
			plan: SharedEnvironmentVariable{
				Value:          types.StringNull(),
				ValueWOVersion: types.Int64Null(),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldUpdateSharedEnvironmentVariableValueWO(tt.state, tt.plan); got != tt.want {
				t.Fatalf("shouldUpdateSharedEnvironmentVariableValueWO() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestSharedEnvironmentVariableSensitiveSemantics(t *testing.T) {
	tests := []struct {
		name                       string
		sensitive                  types.Bool
		wantExplicitlyNonSensitive bool
		wantSensitive              bool
	}{
		{
			name:                       "null is treated as sensitive",
			sensitive:                  types.BoolNull(),
			wantExplicitlyNonSensitive: false,
			wantSensitive:              true,
		},
		{
			name:                       "unknown is treated as sensitive",
			sensitive:                  types.BoolUnknown(),
			wantExplicitlyNonSensitive: false,
			wantSensitive:              true,
		},
		{
			name:                       "true stays sensitive",
			sensitive:                  types.BoolValue(true),
			wantExplicitlyNonSensitive: false,
			wantSensitive:              true,
		},
		{
			name:                       "false is explicitly non-sensitive",
			sensitive:                  types.BoolValue(false),
			wantExplicitlyNonSensitive: true,
			wantSensitive:              false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := SharedEnvironmentVariable{Sensitive: tt.sensitive}

			if got := env.isExplicitlyNonSensitive(); got != tt.wantExplicitlyNonSensitive {
				t.Fatalf("isExplicitlyNonSensitive() = %t, want %t", got, tt.wantExplicitlyNonSensitive)
			}

			if got := env.isSensitive(); got != tt.wantSensitive {
				t.Fatalf("isSensitive() = %t, want %t", got, tt.wantSensitive)
			}
		})
	}
}

func TestSharedEnvironmentVariableHasTarget(t *testing.T) {
	tests := []struct {
		name       string
		target     types.Set
		wantTarget bool
	}{
		{
			name:       "null target",
			target:     types.SetNull(types.StringType),
			wantTarget: false,
		},
		{
			name:       "unknown target",
			target:     types.SetUnknown(types.StringType),
			wantTarget: false,
		},
		{
			name:       "development target present",
			target:     stringSet("development", "preview"),
			wantTarget: true,
		},
		{
			name:       "development target absent",
			target:     stringSet("production", "preview"),
			wantTarget: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := SharedEnvironmentVariable{Target: tt.target}

			got, diags := env.hasTarget(context.Background(), "development")
			if diags.HasError() {
				t.Fatalf("hasTarget() returned diagnostics: %v", diags)
			}

			if got != tt.wantTarget {
				t.Fatalf("hasTarget() = %t, want %t", got, tt.wantTarget)
			}
		})
	}
}
