package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/v3/client"
)

// EnvironmentItem reflects the state terraform stores internally for a project's environment variable.
type EnvironmentItem struct {
	Target               types.Set    `tfsdk:"target"`
	CustomEnvironmentIDs types.Set    `tfsdk:"custom_environment_ids"`
	GitBranch            types.String `tfsdk:"git_branch"`
	Key                  types.String `tfsdk:"key"`
	Value                types.String `tfsdk:"value"`
	ID                   types.String `tfsdk:"id"`
	Sensitive            types.Bool   `tfsdk:"sensitive"`
	Comment              types.String `tfsdk:"comment"`
}

func (e *EnvironmentItem) equal(other *EnvironmentItem) bool {
	return e.Key.ValueString() == other.Key.ValueString() &&
		e.Value.ValueString() == other.Value.ValueString() &&
		e.Target.Equal(other.Target) &&
		e.CustomEnvironmentIDs.Equal(other.CustomEnvironmentIDs) &&
		e.GitBranch.ValueString() == other.GitBranch.ValueString() &&
		e.Sensitive.ValueBool() == other.Sensitive.ValueBool() &&
		e.Comment.ValueString() == other.Comment.ValueString()
}

func (e *EnvironmentItem) toAttrValue() attr.Value {
	return types.ObjectValueMust(EnvVariableElemType.AttrTypes, map[string]attr.Value{
		"id":                     e.ID,
		"key":                    e.Key,
		"value":                  e.Value,
		"target":                 e.Target,
		"custom_environment_ids": e.CustomEnvironmentIDs,
		"git_branch":             e.GitBranch,
		"sensitive":              e.Sensitive,
		"comment":                e.Comment,
	})
}

func (e *EnvironmentItem) toEnvironmentVariableRequest(ctx context.Context) (req client.EnvironmentVariableRequest, diags diag.Diagnostics) {
	var target []string
	diags = e.Target.ElementsAs(ctx, &target, true)
	if diags.HasError() {
		return req, diags
	}
	var customEnvironmentIDs []string
	diags = e.CustomEnvironmentIDs.ElementsAs(ctx, &customEnvironmentIDs, true)
	if diags.HasError() {
		return req, diags
	}

	var envVariableType string
	if e.Sensitive.ValueBool() {
		envVariableType = "sensitive"
	} else {
		envVariableType = "encrypted"
	}

	return client.EnvironmentVariableRequest{
		Key:                  e.Key.ValueString(),
		Value:                e.Value.ValueString(),
		Target:               target,
		CustomEnvironmentIDs: customEnvironmentIDs,
		GitBranch:            e.GitBranch.ValueStringPointer(),
		Type:                 envVariableType,
		Comment:              e.Comment.ValueString(),
	}, nil
}

var EnvVariableElemType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"key":   types.StringType,
		"value": types.StringType,
		"target": types.SetType{
			ElemType: types.StringType,
		},
		"custom_environment_ids": types.SetType{
			ElemType: types.StringType,
		},
		"git_branch": types.StringType,
		"id":         types.StringType,
		"sensitive":  types.BoolType,
		"comment":    types.StringType,
	},
}