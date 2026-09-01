package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestProjectEnvironmentVariablesResourceSchemaRequiresSensitive(t *testing.T) {
	res := newProjectEnvironmentVariablesResource()

	resp := &resource.SchemaResponse{}
	res.Schema(context.Background(), resource.SchemaRequest{}, resp)

	variablesAttr, ok := resp.Schema.Attributes["variables"].(schema.SetNestedAttribute)
	if !ok {
		t.Fatalf("variables attribute has unexpected type: %T", resp.Schema.Attributes["variables"])
	}

	sensitiveAttr, ok := variablesAttr.NestedObject.Attributes["sensitive"].(schema.BoolAttribute)
	if !ok {
		t.Fatalf("variables.sensitive attribute has unexpected type: %T", variablesAttr.NestedObject.Attributes["sensitive"])
	}

	assertBoolRequired(t, sensitiveAttr, "variables.sensitive")
}

func TestProjectResourceEnvironmentSchemaRequiresSensitive(t *testing.T) {
	res := newProjectResource()

	resp := &resource.SchemaResponse{}
	res.Schema(context.Background(), resource.SchemaRequest{}, resp)

	environmentAttr, ok := resp.Schema.Attributes["environment"].(schema.SetNestedAttribute)
	if !ok {
		t.Fatalf("environment attribute has unexpected type: %T", resp.Schema.Attributes["environment"])
	}

	sensitiveAttr, ok := environmentAttr.NestedObject.Attributes["sensitive"].(schema.BoolAttribute)
	if !ok {
		t.Fatalf("environment.sensitive attribute has unexpected type: %T", environmentAttr.NestedObject.Attributes["sensitive"])
	}

	assertBoolRequired(t, sensitiveAttr, "environment.sensitive")
}

func TestEnvironmentItemSensitiveSemantics(t *testing.T) {
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
			env := EnvironmentItem{Sensitive: tt.sensitive}

			if got := env.isExplicitlyNonSensitive(); got != tt.wantExplicitlyNonSensitive {
				t.Fatalf("isExplicitlyNonSensitive() = %t, want %t", got, tt.wantExplicitlyNonSensitive)
			}

			if got := env.isSensitive(); got != tt.wantSensitive {
				t.Fatalf("isSensitive() = %t, want %t", got, tt.wantSensitive)
			}
		})
	}
}

func TestEnvironmentItemHasTarget(t *testing.T) {
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
			env := EnvironmentItem{Target: tt.target}

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

func TestEnvironmentItemToEnvironmentVariableRequestTreatsUnsetSensitiveAsSensitive(t *testing.T) {
	env := EnvironmentItem{
		Target:               types.SetNull(types.StringType),
		CustomEnvironmentIDs: stringSet("ce_123"),
		Key:                  types.StringValue("EXAMPLE"),
		Value:                types.StringValue("value"),
		Sensitive:            types.BoolNull(),
	}

	req, diags := env.toEnvironmentVariableRequest(context.Background())
	if diags.HasError() {
		t.Fatalf("toEnvironmentVariableRequest() returned diagnostics: %v", diags)
	}

	if req.Type != "sensitive" {
		t.Fatalf("toEnvironmentVariableRequest().Type = %q, want %q", req.Type, "sensitive")
	}

	if len(req.CustomEnvironmentIDs) != 1 || req.CustomEnvironmentIDs[0] != "ce_123" {
		t.Fatalf("toEnvironmentVariableRequest().CustomEnvironmentIDs = %v, want [ce_123]", req.CustomEnvironmentIDs)
	}
}

func assertBoolRequired(t *testing.T, attr schema.BoolAttribute, label string) {
	t.Helper()

	if !attr.Required {
		t.Fatalf("%s should be required", label)
	}
	if attr.Optional {
		t.Fatalf("%s should not be optional", label)
	}
	if attr.Computed {
		t.Fatalf("%s should not be computed", label)
	}
	if attr.Default != nil {
		t.Fatalf("%s should not have a default", label)
	}
}
